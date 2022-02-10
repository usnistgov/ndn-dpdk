package fibdef

import (
	binutils "github.com/jfoster/binary-utilities"
	"github.com/pkg/math"
)

// Limits and defaults.
const (
	MinCapacity     = 1<<8 - 1
	MaxCapacity     = 1<<31 - 1
	DefaultCapacity = 1<<16 - 1

	MinStartDepth     = 2
	MaxStartDepth     = 17
	DefaultStartDepth = 8
)

// Config contains FIB configuration.
type Config struct {
	Capacity   int `json:"capacity,omitempty"`   // Capacity.
	NBuckets   int `json:"nBuckets,omitempty"`   // Hashtable buckets.
	StartDepth int `json:"startDepth,omitempty"` // 'M' in 2-stage LPM algorithm.
}

// ApplyDefaults applies defaults.
func (cfg *Config) ApplyDefaults() {
	if cfg.Capacity == 0 {
		cfg.Capacity = DefaultCapacity
	} else {
		cfg.Capacity = math.MinInt(math.MaxInt(MinCapacity, cfg.Capacity), MaxCapacity)
	}

	if cfg.NBuckets <= 0 {
		cfg.NBuckets = (cfg.Capacity + 1) / 2
	}
	cfg.NBuckets = int(binutils.NearPowerOfTwo(int64(cfg.NBuckets)))

	if cfg.StartDepth == 0 {
		cfg.StartDepth = DefaultStartDepth
	} else {
		cfg.StartDepth = math.MinInt(math.MaxInt(MinStartDepth, cfg.StartDepth), MaxStartDepth)
	}
}
