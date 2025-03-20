package bdev

/*
#include "../../csrc/dpdk/bdev.h"
#include <spdk/nvme.h>
*/
import "C"
import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/pciaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
)

// NvmeNamespace represents an NVMe namespace.
type NvmeNamespace struct {
	info *Info
	nvme *Nvme
}

var _ interface {
	Device
	withWriteMode
} = &NvmeNamespace{}

// DevInfo implements Device interface.
func (nn *NvmeNamespace) DevInfo() *Info {
	return nn.info
}

// Controller returns NVMe controller.
func (nn *NvmeNamespace) Controller() *Nvme {
	return nn.nvme
}

func (nn *NvmeNamespace) writeMode() WriteMode {
	supported, dwordAlign := nn.Controller().SglSupport()
	if supported {
		if dwordAlign {
			return WriteModeDwordAlign
		}
		return WriteModeSimple
	}
	return WriteModeContiguous
}

// Nvme represents an NVMe controller.
type Nvme struct {
	// Namespaces is a list of NVMe namespaces as block devices.
	Namespaces []*NvmeNamespace

	pciAddr pciaddr.PCIAddress
	flags   uint64
}

// ControllerName returns NVMe controller name.
func (nvme *Nvme) ControllerName() string {
	return "nvme" + nvme.pciAddr.String()
}

// SglSupport reports whether NVMe controller supports scatter-gather lists and whether it requires dword alignment.
func (nvme *Nvme) SglSupport() (supported, dwordAlign bool) {
	return nvme.flags&C.SPDK_NVME_CTRLR_SGL_SUPPORTED != 0,
		nvme.flags&C.SPDK_NVME_CTRLR_SGL_REQUIRES_DWORD_ALIGNMENT != 0
}

// Close detaches the NVMe controller.
func (nvme *Nvme) Close() error {
	return deleteByName("bdev_nvme_detach_controller", nvme.ControllerName())
}

// AttachNvme attaches block devices on an NVMe controller.
func AttachNvme(pciAddr pciaddr.PCIAddress) (nvme *Nvme, e error) {
	nvme = &Nvme{pciAddr: pciAddr}

	var trid C.struct_spdk_nvme_transport_id
	trid.trtype = C.SPDK_NVME_TRANSPORT_PCIE
	copy(cptr.AsByteSlice(trid.traddr[:]), pciAddr.String())
	ctrlr := C.spdk_nvme_connect(&trid, nil, 0)
	if ctrlr == nil {
		return nil, errors.New("spdk_nvme_connect error")
	}
	nvme.flags = uint64(C.spdk_nvme_ctrlr_get_flags(ctrlr))
	C.spdk_nvme_detach(ctrlr)

	initBdevLib()
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

	for _, nn := range namespaces {
		nvme.Namespaces = append(nvme.Namespaces, &NvmeNamespace{
			info: mustFind(nn),
			nvme: nvme,
		})
	}
	return nvme, nil
}
