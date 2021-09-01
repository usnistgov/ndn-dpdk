package fileserver

import (
	"errors"
	"fmt"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_FILESERVER_ENUM_H -out=../../csrc/fileserver/enum.h .

// Limits and defaults.
const (
	MaxMounts = 8

	MinSegmentLen     = 64
	MaxSegmentLen     = 16384
	DefaultSegmentLen = 4096

	MinUringCapacity     = 64
	DefaultUringCapacity = 4096

	MinOpenFds     = 16
	MaxOpenFds     = 16384
	DefaultOpenFds = 256

	MinKeepFds     = 4
	MaxKeepFds     = 16384
	DefaultKeepFds = 64

	_ = "enumgen::FileServer"
)

// Config contains FileServer configuration.
type Config struct {
	NThreads int                  `json:"nThreads,omitempty"`
	RxQueue  iface.PktQueueConfig `json:"rxQueue,omitempty"`

	// Mounts is a list of name prefix and filesystem path.
	// There must be between 1 and MaxMounts entries.
	// Prefixes should not overlap.
	Mounts []Mount `json:"mounts"`

	// SegmentLen is maximum TLV-LENGTH of Data Content payload.
	// This value must be set consistently in every producer of the same name prefix.
	SegmentLen int `json:"segmentLen,omitempty"`

	// UringCapacity is io_uring queue size.
	UringCapacity int `json:"uringCapacity,omitempty"`

	// OpenFds is the limit of open file descriptors (including KeepFds) per thread.
	// This is used to calculate data structure sizes; it is not a hard limit.
	// You must also set `ulimit -n` or systemd `LimitNOFILE=` appropriately.
	OpenFds int `json:"openFds,omitempty"`

	// KeepFds is the number of unused file descriptor per thread.
	// A file descriptor is unused if no I/O operation is ongoing on the file.
	// Keeping them open can speed up subsequent requests referencing the same file.
	KeepFds int `json:"keepFds,omitempty"`

	payloadHeadroom int
}

// Validate applies defaults and validates the configuration.
func (cfg *Config) Validate() error {
	cfg.NThreads = math.MaxInt(1, cfg.NThreads)

	cfg.RxQueue.DisableCoDel = true

	if len(cfg.Mounts) == 0 {
		return errors.New("no mount specified")
	}
	if len(cfg.Mounts) > MaxMounts {
		return fmt.Errorf("cannot add more than %d mounts", MaxMounts)
	}
	for i, m := range cfg.Mounts {
		if m.Prefix.Length() > ndni.NameMaxLength {
			return fmt.Errorf("mounts[%d].prefix cannot exceed %d octets", i, ndni.NameMaxLength)
		}
		for _, comp := range m.Prefix {
			if comp.Type != an.TtGenericNameComponent {
				return fmt.Errorf("mounts[%d].prefix must consist of GenericNameComponents", i)
			}
		}
	}

	if cfg.SegmentLen == 0 {
		cfg.SegmentLen = DefaultSegmentLen
	}
	if cfg.SegmentLen < MinSegmentLen || cfg.SegmentLen > MaxSegmentLen {
		return fmt.Errorf("segmentLen out of range [%d:%d]", MinSegmentLen, MaxSegmentLen)
	}

	cfg.UringCapacity = ringbuffer.AlignCapacity(cfg.UringCapacity, MinUringCapacity, DefaultUringCapacity)

	if cfg.OpenFds == 0 {
		cfg.OpenFds = DefaultOpenFds
	}
	if cfg.OpenFds < MinOpenFds || cfg.OpenFds > MaxOpenFds {
		return fmt.Errorf("openFds out of range [%d:%d]", MinOpenFds, MaxOpenFds)
	}

	if cfg.KeepFds == 0 {
		cfg.KeepFds = DefaultKeepFds
	}
	if cfg.KeepFds < MinKeepFds || cfg.KeepFds > MaxKeepFds {
		return fmt.Errorf("keepFds out of range [%d:%d]", MinKeepFds, MaxKeepFds)
	}

	if cfg.OpenFds <= cfg.KeepFds {
		return errors.New("openFds must be greater than keepFds")
	}

	return nil
}

func (cfg *Config) checkPayloadMempool(segmentLen int) error {
	tpl := ndni.PayloadMempool.Config()

	suggestCapacity := cfg.UringCapacity * cfg.NThreads
	if tpl.Capacity+1 < suggestCapacity {
		// tpl.Capacity+1 so that (2^n-1) is accepted when suggestion is (2^n)
		logger.Warn("PAYLOAD capacity too small for fileserver",
			zap.Int("configured-capacity", tpl.Capacity),
			zap.Int("suggested-capacity", suggestCapacity),
		)
	}

	if tpl.Dataroom-pktmbuf.DefaultHeadroom < int(sizeofFileServerFd) {
		return fmt.Errorf("PAYLOAD dataroom %d too small for struct FileServerFd; increase PAYLOAD dataroom to %d",
			tpl.Dataroom, sizeofFileServerFd)
	}

	suggest := pktmbuf.DefaultHeadroom + ndni.NameMaxLength + segmentLen + ndni.DataEncNullSigLen + 64
	if tpl.Dataroom < suggest {
		logger.Warn("PAYLOAD dataroom too small for configured segmentLen, Interests with long names may be dropped",
			zap.Int("configured-dataroom", tpl.Dataroom),
			zap.Int("configured-segmentlen", segmentLen),
			zap.Int("suggested-dataroom", suggest),
		)
	}

	cfg.payloadHeadroom = tpl.Dataroom - ndni.DataEncNullSigLen - segmentLen
	if cfg.payloadHeadroom < pktmbuf.DefaultHeadroom {
		return fmt.Errorf("PAYLOAD dataroom %d too small for segmentLen %d; increase PAYLOAD dataroom to %d",
			tpl.Dataroom, segmentLen, suggest)
	}
	return nil
}

// Mount defines a mapping between name prefix and filesystem path.
type Mount struct {
	Prefix ndn.Name `json:"prefix"`
	Path   string   `json:"path"`
	dfd    *int
}

func (m *Mount) openDirectory() error {
	m.closeDirectory()

	dfd, e := unix.Open(m.Path, unix.O_RDONLY|unix.O_DIRECTORY, 0)
	if e != nil {
		return fmt.Errorf("open(%s,O_DIRECTORY) %w", m.Path, e)
	}
	m.dfd = &dfd
	return nil
}

func (m *Mount) closeDirectory() (e error) {
	if m.dfd != nil {
		e = unix.Close(*m.dfd)
	}
	m.dfd = nil
	return e
}
