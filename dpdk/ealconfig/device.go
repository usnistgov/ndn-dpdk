package ealconfig

// DeviceConfig contains device related configuration.
type DeviceConfig struct {
	// PciDevices is an allowlist of PCI devices to enable.
	// This may include Ethernet adapters, NVMe storage controllers, etc.
	// Each should be a PCI address.
	// If omitted, PCI bus is disabled.
	PciDevices []PciAddress `json:"pciDevices,omitempty"`

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

	for _, dev := range cfg.PciDevices {
		args = append(args, "-w", dev.String())
	}

	for _, dev := range cfg.VirtualDevices {
		args = append(args, "--vdev", dev)
	}

	return args, nil
}
