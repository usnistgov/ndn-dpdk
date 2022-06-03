package ealconfig

import (
	"strings"

	"github.com/usnistgov/ndn-dpdk/core/hwinfo"
)

// PmdPath is the location of DPDK drivers.
// This is assigned to C.RTE_EAL_PMD_PATH by ealinit package.
var PmdPath string

// DeviceConfig contains device related configuration.
type DeviceConfig struct {
	// IovaMode selects IO Virtual Addresses mode.
	// Possible values are "PA" and "VA".
	// Default is letting DPDK decide automatically based on loaded drivers and kernel options.
	//
	// Some DPDK drivers may require a particular mode, and will not work in the other mode.
	// Read "Memory in DPDK Part 2: Deep Dive into IOVA" for how to choose a mode:
	// https://www.intel.com/content/www/us/en/developer/articles/technical/memory-in-dpdk-part-2-deep-dive-into-iova.html
	IovaMode string `json:"iovaMode,omitempty"`

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

	switch cfg.IovaMode {
	case "PA", "VA":
		args = append(args, "--iova-mode", strings.ToLower(cfg.IovaMode))
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
