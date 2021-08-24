package fileserver

import (
	"errors"
	"fmt"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
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

	_ = "enumgen::FileServer"
)

// Error conditions.
var (
	ErrNoMount       = errors.New("no mount specified")
	ErrTooManyMounts = fmt.Errorf("cannot add more than %d mounts", MaxMounts)
	ErrPrefixTooLong = fmt.Errorf("prefix cannot exceed %d octets", ndni.NameMaxLength)
	ErrSegmentLen    = fmt.Errorf("segmentLen out of range [%d:%d]", MinSegmentLen, MaxSegmentLen)
)

// Config contains FileServer configuration.
type Config struct {
	NThreads int                  `json:"nThreads,omitempty"`
	RxQueue  iface.PktQueueConfig `json:"rxQueue,omitempty"`

	// Mounts is a list of name prefix and filesystem path.
	// There must be at least 1 and at most MaxMounts entries.
	// Prefixes should not overlap.
	Mounts []Mount

	// SegmentLen is maximum TLV-LENGTH of Data Content payload.
	// This value must be set consistently in every producer of the same name prefix.
	SegmentLen int `json:"segmentLen,omitempty"`

	// UringCapacity is io_uring queue size.
	// Default is 4096.
	UringCapacity int `json:"uringCapacity,omitempty"`

	payloadHeadroom int
}

func (cfg *Config) applyDefaults() {
	cfg.RxQueue.DisableCoDel = true
	cfg.NThreads = math.MaxInt(1, cfg.NThreads)
	if cfg.SegmentLen == 0 {
		cfg.SegmentLen = DefaultSegmentLen
	}
	cfg.UringCapacity = ringbuffer.AlignCapacity(cfg.UringCapacity, 64, 4096)
}

func (cfg Config) validate() error {
	if len(cfg.Mounts) == 0 {
		return ErrNoMount
	}
	if len(cfg.Mounts) > MaxMounts {
		return ErrTooManyMounts
	}
	for _, m := range cfg.Mounts {
		if m.Prefix.Length() > ndni.NameMaxLength {
			return ErrPrefixTooLong
		}
	}
	if cfg.SegmentLen < MinSegmentLen || cfg.SegmentLen > MaxSegmentLen {
		return ErrSegmentLen
	}
	return nil
}

func (cfg Config) checkDirectMempoolDataroom() {
	dataroom := pktmbuf.Direct.Config().Dataroom
	suggest := pktmbuf.DefaultHeadroom + ndni.NameMaxLength + 64 + unix.PathMax
	if dataroom < suggest {
		logger.Warn("DIRECT dataroom too small, Interests with long names may be dropped",
			zap.Int("configured-dataroom", dataroom),
			zap.Int("suggested-dataroom", suggest),
		)
	}
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
