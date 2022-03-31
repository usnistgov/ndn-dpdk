package bdev

import (
	"github.com/usnistgov/ndn-dpdk/core/pciaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
)

// NvmeNamespace represents an NVMe namespace.
type NvmeNamespace struct {
	*Info
}

var _ Device = (*NvmeNamespace)(nil)

// Nvme represents an NVMe controller.
type Nvme struct {
	// Namespaces is a list of NVMe namespaces as block devices.
	Namespaces []*NvmeNamespace

	pciAddr pciaddr.PCIAddress
}

// ControllerName returns NVMe controller name.
func (nvme *Nvme) ControllerName() string {
	return "nvme" + nvme.pciAddr.String()
}

// Close detaches the NVMe controller.
func (nvme *Nvme) Close() error {
	return deleteByName("bdev_nvme_detach_controller", nvme.ControllerName())
}

// AttachNvme attaches block devices on an NVMe controller.
func AttachNvme(pciAddr pciaddr.PCIAddress) (nvme *Nvme, e error) {
	initBdevLib()
	nvme = &Nvme{pciAddr: pciAddr}
	args := struct {
		Name   string `json:"name"`
		TrType string `json:"trtype"`
		TrAddr string `json:"traddr"`
	}{
		Name:   nvme.ControllerName(),
		TrType: "pcie",
		TrAddr: pciAddr.String(),
	}

	var namespaces []string
	if e = spdkenv.RPC("bdev_nvme_attach_controller", args, &namespaces); e != nil {
		return nil, e
	}

	for _, namespace := range namespaces {
		nvme.Namespaces = append(nvme.Namespaces, &NvmeNamespace{mustFind(namespace)})
	}
	return nvme, nil
}
