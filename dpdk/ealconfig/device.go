package ealconfig

import (
	"github.com/usnistgov/ndn-dpdk/core/hwinfo"
)

// PmdPath is the location of DPDK drivers.
// This is assigned to C.RTE_EAL_PMD_PATH by ealinit package.
var PmdPath string

// DeviceConfig contains device related configuration.
type DeviceConfig struct {
	// Drivers is a list of shared object files or directories containing them.
	// Default is to include all DPDK drivers.
	//
	// If this is overridden, you must include these drivers:
	// - librte_crypto_openssl.so
	// - librte_mempool_ring.so
	// - librte_net_af_packet.so
	// - librte_net_memif.so
	// Not loading these drivers will likely cause NDN-DPDK activation failure.
	Drivers []string `json:"drivers,omitempty"`

	// DisablePCI disables the PCI bus.
	DisablePCI bool `json:"disablePCI,omitempty"`

	// DeviceFlags is device-related flags passed to DPDK.
	// This replaces all other options.
	DeviceFlags string `json:"deviceFlags,omitempty"`
}

func (cfg DeviceConfig) args(hwInfo hwinfo.Provider) (args []string, e error) {
	if cfg.DeviceFlags != "" {
		return shellSplit("deviceFlags", cfg.DeviceFlags)
	}

	switch {
	case len(cfg.Drivers) > 0:
		for _, drvPath := range cfg.Drivers {
			args = append(args, "-d", drvPath)
		}
	case PmdPath != "":
		args = append(args, "-d", PmdPath)
	default:
		logger.Fatal("PmdPath is unassigned")
	}

	if cfg.DisablePCI {
		args = append(args, "--no-pci")
	} else {
		args = append(args, "-a", "00:00.0")
	}

	return args, nil
}
