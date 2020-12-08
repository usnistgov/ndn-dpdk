package ealconfig

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

	// PciDevices is an allowlist of PCI devices to enable.
	// This may include Ethernet adapters, NVMe storage controllers, etc.
	// Each should be a PCI address.
	PciDevices []PciAddress `json:"pciDevices,omitempty"`

	// AllPciDevices enables all PCI devices.
	// If AllPciDevices is false and PciDevices is empty, the PCI bus is disabled.
	AllPciDevices bool `json:"allPciDevices,omitempty"`

	// VirtualDevices is a list of virtual devices.
	// Each should be a device argument for DPDK --vdev flag.
	VirtualDevices []string `json:"virtualDevices,omitempty"`

	// DeviceFlags is device-related flags passed to DPDK.
	// This replaces all other options.
	DeviceFlags string `json:"deviceFlags,omitempty"`
}

func (cfg DeviceConfig) args(req Request, hwInfo HwInfoSource) (args []string, e error) {
	if cfg.DeviceFlags != "" {
		return shellSplit("DeviceFlags", cfg.DeviceFlags)
	}

	switch {
	case len(cfg.Drivers) > 0:
		for _, drvPath := range cfg.Drivers {
			args = append(args, "-d", drvPath)
		}
	case PmdPath != "":
		args = append(args, "-d", PmdPath)
	default:
		log.Fatal("PmdPath is unassigned")
	}

	switch {
	case cfg.AllPciDevices:
	case len(cfg.PciDevices) == 0:
		args = append(args, "--no-pci")
	default:
		for _, dev := range cfg.PciDevices {
			args = append(args, "-w", dev.String())
		}
	}

	for _, dev := range cfg.VirtualDevices {
		args = append(args, "--vdev", dev)
	}

	return args, nil
}
