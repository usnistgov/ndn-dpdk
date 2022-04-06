package ndt

import (
	binutils "github.com/jfoster/binary-utilities"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/zyedidia/generic"
)

// Limits and defaults.
const (
	MinPrefixLen     = 1
	MaxPrefixLen     = ndni.PNameCachedComponents
	DefaultPrefixLen = 2

	MinCapacity     = 1 << 4
	MaxCapacity     = 1 << 31
	DefaultCapacity = 1 << 16

	MinSampleInterval     = 1 << 0
	MaxSampleInterval     = 1 << 30
	DefaultSampleInterval = 1 << 10
)

// Config contains NDT configuration.
type Config struct {
	// PrefixLen is the number of name components considered in NDT lookup.
	//
	// If this value is zero, it defaults to DefaultPrefixLen.
	// Otherwise, it is clamped between MinPrefixLen and MaxPrefixLen.
	PrefixLen int `json:"prefixLen,omitempty" gqldesc:"Number of name components considered in NDT lookup."`

	// Capacity is the number of NDT entries.
	//
	// If this value is zero, it defaults to DefaultCapacity.
	// Otherwise, it is clamped between MinCapacity and MaxCapacity, and adjusted up to the next power of 2.
	Capacity int `json:"capacity,omitempty" gqldesc:"Number of NDT entries."`

	// SampleInterval indicates how often per-entry counters are incremented within a lookup thread.
	//
	// If this value is zero, it defaults to DefaultSampleInterval.
	// Otherwise, it is clamped between MinSampleInterval and MaxSampleInterval, and adjusted up to the next power of 2.
	SampleInterval int `json:"sampleInterval,omitempty" gqldesc:"How often per-entry counters are incremented."`
}

func (c *Config) applyDefaults() {
	if c.PrefixLen == 0 {
		c.PrefixLen = DefaultPrefixLen
	} else {
		c.PrefixLen = generic.Clamp(c.PrefixLen, MinPrefixLen, MaxPrefixLen)
	}

	if c.Capacity == 0 {
		c.Capacity = DefaultCapacity
	} else {
		c.Capacity = generic.Clamp(c.Capacity, MinCapacity, MaxCapacity)
	}
	c.Capacity = int(binutils.NextPowerOfTwo(int64(c.Capacity)))

	if c.SampleInterval == 0 {
		c.SampleInterval = DefaultSampleInterval
	} else {
		c.SampleInterval = generic.Clamp(c.PrefixLen, MinSampleInterval, MaxSampleInterval)
	}
	c.SampleInterval = int(binutils.NextPowerOfTwo(int64(c.SampleInterval)))
}

func (c Config) indexMask() uint64 {
	return uint64(c.Capacity - 1)
}
