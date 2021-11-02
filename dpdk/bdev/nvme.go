package bdev

/*
#include "../../csrc/core/common.h"
#include <spdk/nvme.h>

extern bool go_nvmeProbed(uintptr_t ctx, struct spdk_nvme_transport_id* trid, struct spdk_nvme_ctrlr_opts* opts);

static int c_spdk_nvme_probe(uintptr_t ctx)
{
	return spdk_nvme_probe(NULL, (void*)ctx, (spdk_nvme_probe_cb)go_nvmeProbed, NULL, NULL);
}
*/
import "C"
import (
	"errors"
	"runtime/cgo"

	"github.com/usnistgov/ndn-dpdk/core/pciaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
)

type listNvmesResult struct {
	nvmes []pciaddr.PCIAddress
}

// ListNvmes returns a list of NVMe drives.
func ListNvmes() (nvmes []pciaddr.PCIAddress, e error) {
	var result listNvmesResult
	ctx := cgo.NewHandle(&result)
	defer ctx.Delete()
	res := eal.CallMain(func() int { return int(C.c_spdk_nvme_probe(C.uintptr_t(ctx))) }).(int)
	if res != 0 {
		return nil, errors.New("spdk_nvme_probe error")
	}
	return result.nvmes, nil
}

//export go_nvmeProbed
func go_nvmeProbed(ctx C.uintptr_t, trid *C.struct_spdk_nvme_transport_id, opts *C.struct_spdk_nvme_ctrlr_opts) C.bool {
	pciAddr := pciaddr.MustParse(C.GoString(&trid.traddr[0]))
	result := cgo.Handle(ctx).Value().(*listNvmesResult)
	result.nvmes = append(result.nvmes, pciAddr)
	return C.bool(false)
}

// Nvme represents block devices on an NVMe drives.
type Nvme struct {
	// Namespaces is a list of NVMe namespaces as block devices.
	Namespaces []*Info

	pciAddr pciaddr.PCIAddress
}

func (nvme *Nvme) getName() string {
	return "nvme" + nvme.pciAddr.String()
}

// AttachNvme attaches block devices on an NVMe drives.
func AttachNvme(pciAddr pciaddr.PCIAddress) (nvme *Nvme, e error) {
	initBdevLib()
	nvme = new(Nvme)
	nvme.pciAddr = pciAddr
	var args bdevNvmeAttachControllerArgs
	args.Name = nvme.getName()
	args.TrType = "pcie"
	args.TrAddr = pciAddr.String()

	var namespaces []string
	if e = spdkenv.RPC("bdev_nvme_attach_controller", args, &namespaces); e != nil {
		return nil, e
	}

	for _, namespace := range namespaces {
		nvme.Namespaces = append(nvme.Namespaces, Find(namespace))
	}
	return nvme, nil
}

// Close detaches the NVMe drives.
func (nvme *Nvme) Close() error {
	var args bdevNvmeDetachControllerArgs
	args.Name = nvme.getName()
	var ok bool
	return spdkenv.RPC("bdev_nvme_detach_controller", args, &ok)
}

type bdevNvmeAttachControllerArgs struct {
	Name   string `json:"name"`
	TrType string `json:"trtype"`
	TrAddr string `json:"traddr"`
}

type bdevNvmeDetachControllerArgs struct {
	Name string `json:"name"`
}
