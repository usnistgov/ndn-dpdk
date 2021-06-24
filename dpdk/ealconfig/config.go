// Package ealconfig prepares EAL parameters.
package ealconfig

import (
	"github.com/usnistgov/ndn-dpdk/core/hwinfo"
	"github.com/usnistgov/ndn-dpdk/core/logging"
)

var logger = logging.New("ealconfig")

type section interface {
	args(hwInfo hwinfo.Provider) (args []string, e error)
}

// Config contains EAL configuration.
type Config struct {
	LCoreConfig
	MemoryConfig
	DeviceConfig

	// ExtraFlags is additional flags passed to DPDK.
	ExtraFlags string `json:"extraFlags,omitempty"`

	// Flags is all flags passed to DPDK.
	// This replaces all other options.
	Flags string `json:"flags,omitempty"`
}

// Args validates the configuration and constructs EAL arguments.
func (cfg Config) Args(hwInfo hwinfo.Provider) (args []string, e error) {
	if cfg.Flags != "" {
		return shellSplit("Flags", cfg.Flags)
	}
	if hwInfo == nil {
		hwInfo = hwinfo.Default
	}

	for _, sec := range []section{cfg.LCoreConfig, cfg.MemoryConfig, cfg.DeviceConfig} {
		a, e := sec.args(hwInfo)
		if e != nil {
			return nil, e
		}
		args = append(args, a...)
	}

	if cfg.ExtraFlags != "" {
		a, e := shellSplit("ExtraFlags", cfg.ExtraFlags)
		if e != nil {
			return nil, e
		}
		args = append(args, a...)
	}
	return args, nil
}
