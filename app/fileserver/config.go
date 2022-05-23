package fileserver

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/zyedidia/generic"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_FILESERVER_ENUM_H -out=../../csrc/fileserver/enum.h .
//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_FILESERVER_AN_H -out=../../csrc/fileserver/an.h ../../ndn/rdr/ndn6file

// Limits and defaults.
const (
	MaxMounts   = 8
	MaxIovecs   = 1
	MaxLsResult = 262144

	MinSegmentLen     = 64
	MaxSegmentLen     = 16384
	DefaultSegmentLen = 4096

	MinUringCapacity     = 256
	MaxUringCapacity     = 32768 // KERN_MAX_ENTRIES in liburing
	DefaultUringCapacity = 4096

	_                           = "enumgen+2"
	DefaultUringCongestionThres = 0.7
	DefaultUringWaitThres       = 0.9

	MinOpenFds     = 16
	MaxOpenFds     = 16384
	DefaultOpenFds = 256

	MinKeepFds     = 4
	MaxKeepFds     = 16384
	DefaultKeepFds = 64

	DefaultStatValidityMilliseconds = 10 * 1000 // 10 seconds

	EstimatedMetadataSize = 4 + // NameTL, excluding NameV
		2 + 10 + // FinalBlockId
		7*10 // NNI fields

	MetadataFreshness = 1

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
	SegmentLen int `json:"segmentLen,omitempty" gqldesc:"Maximum TLV-LENGTH of Data Content payload."`

	// UringCapacity is io_uring submission queue size.
	// When pending I/O operations exceed 50% capacity, congestion marks start to appear on Data packets.
	// When pending I/O operations exceed 75% capacity, submissions will block waiting for completions.
	UringCapacity int `json:"uringCapacity,omitempty" gqldesc:"uring submission queue size."`

	// UringCongestionThres is the uring occupancy threshold to start inserting congetion marks.
	// If uring occupancy ratio exceeds this threshold, congestion marks are added to some outgoing Data packets
	// This must be between 0.0 (exclusive) and 1.0 (exclusive); it should be smaller than UringWaitThres.
	UringCongestionThres float64 `json:"uringCongestionThres,omitempty" gqldesc:"uring occupancy threshold to start inserting congestion marks."`

	// UringWaitThres is the uring occupancy threshold to start waiting for completions.
	// If uring occupancy ratio exceeds this threshold, uring submission will block and wait for completions.
	// This must be between 0.0 (exclusive) and 1.0 (exclusive).
	UringWaitThres float64 `json:"uringWaitThres,omitempty" gqldesc:"uring occupancy threshold to start waiting for completions."`

	// OpenFds is the limit of open file descriptors (including KeepFds) per thread.
	// You must also set `ulimit -n` or systemd `LimitNOFILE=` appropriately.
	OpenFds int `json:"openFds,omitempty" gqldesc:"Maximum open file descriptors per thread."`

	// KeepFds is the number of unused file descriptors per thread.
	// A file descriptor is unused if no I/O operation is ongoing on the file.
	// Keeping them open can speed up subsequent requests referencing the same file.
	KeepFds int `json:"keepFds,omitempty" gqldesc:"Maximum unused file descriptors per thread."`

	// StatValidity is the validity period of statx result.
	StatValidity nnduration.Nanoseconds `json:"statValidity,omitempty" gqldesc:"statx result validity period."`

	payloadHeadroom       int
	uringCongestionLbound int
	uringWaitLbound       int
}

// Validate applies defaults and validates the configuration.
func (cfg *Config) Validate() error {
	cfg.NThreads = generic.Max(1, cfg.NThreads)

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

	cfg.UringCapacity = ringbuffer.AlignCapacity(cfg.UringCapacity, MinUringCapacity, DefaultUringCapacity, MaxUringCapacity)
	cfg.uringCongestionLbound = cfg.adjustUringThres(&cfg.UringCongestionThres, DefaultUringCongestionThres)
	cfg.uringWaitLbound = cfg.adjustUringThres(&cfg.UringWaitThres, DefaultUringWaitThres)

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

	if e := cfg.checkPayloadMempool(); e != nil {
		return e
	}

	return nil
}

func (cfg Config) adjustUringThres(thres *float64, dflt float64) (lbound int) {
	if math.IsNaN(*thres) || *thres <= 0.0 || *thres >= 1.0 {
		*thres = dflt
	}
	lbound = int(float64(cfg.UringCapacity) * (*thres))
	return generic.Clamp(lbound, iface.MaxBurstSize, cfg.UringCapacity-iface.MaxBurstSize)
}

func (cfg *Config) checkPayloadMempool() error {
	tpl := ndni.PayloadMempool.Config()

	suggestCapacity := cfg.UringCapacity * cfg.NThreads
	if tpl.Capacity+1 < suggestCapacity {
		// tpl.Capacity+1 so that (2^n-1) is accepted when suggestion is (2^n)
		logger.Warn("PAYLOAD capacity too small for fileserver",
			zap.Int("configured-capacity", tpl.Capacity),
			zap.Int("suggested-capacity", suggestCapacity),
		)
	}

	suggest := pktmbuf.DefaultHeadroom + ndni.NameMaxLength +
		generic.Max(cfg.SegmentLen, ndni.NameMaxLength+EstimatedMetadataSize) + ndni.DataEncNullSigLen + 64
	if tpl.Dataroom < suggest {
		logger.Warn("PAYLOAD dataroom too small for configured segmentLen, Interests with long names may be dropped",
			zap.Int("configured-dataroom", tpl.Dataroom),
			zap.Int("configured-segmentlen", cfg.SegmentLen),
			zap.Int("suggested-dataroom", suggest),
		)
	}

	cfg.payloadHeadroom = tpl.Dataroom - ndni.DataEncNullSigLen - generic.Max(cfg.SegmentLen, EstimatedMetadataSize)
	if cfg.payloadHeadroom < pktmbuf.DefaultHeadroom {
		return fmt.Errorf("PAYLOAD dataroom %d too small for segmentLen %d; increase PAYLOAD dataroom to %d",
			tpl.Dataroom, cfg.SegmentLen, suggest)
	}
	return nil
}

func (cfg Config) tscStatValidity() int64 {
	return eal.ToTscDuration(cfg.StatValidity.DurationOr(nnduration.Nanoseconds(DefaultStatValidityMilliseconds * time.Millisecond)))
}

// Mount defines a mapping between name prefix and filesystem path.
type Mount struct {
	Prefix ndn.Name `json:"prefix" gqldesc:"NDN name prefix."`
	Path   string   `json:"path" gqldesc:"Filesystem path."`
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
