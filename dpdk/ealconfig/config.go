// Package ealconfig prepares EAL parameters.
package ealconfig

import (
	"github.com/usnistgov/ndn-dpdk/core/logging"
)

var logger = logging.New("ealconfig")

// Request contains requirements of the activating application.
type Request struct {
	// MinLCores is the minimum required number of lcores.
	// This is processed by LCoreConfig.
	//
	// If there are fewer processor cores than MinLCores, lcores will be created as threads
	// floating among available cores, resulting in lower performance.
	MinLCores int
}

type section interface {
	args(req Request, hwInfo HwInfoSource) (args []string, e error)
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
func (cfg Config) Args(req Request, hwInfo HwInfoSource) (args []string, e error) {
	if cfg.Flags != "" {
		return shellSplit("Flags", cfg.Flags)
	}
	if hwInfo == nil {
		hwInfo = defaultHwInfoSource{}
	}

	for _, sec := range []section{cfg.LCoreConfig, cfg.MemoryConfig, cfg.DeviceConfig} {
		a, e := sec.args(req, hwInfo)
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
