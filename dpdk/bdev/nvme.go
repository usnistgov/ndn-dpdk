package bdev

/*
#include "../../csrc/core/common.h"
#include <spdk/nvme.h>

extern bool go_nvmeProbed(void* ctx, struct spdk_nvme_transport_id* trid, struct spdk_nvme_ctrlr_opts* opts);
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
)

type listNvmesResult struct {
	nvmes []ealconfig.PCIAddress
}

// ListNvmes returns a list of NVMe drives.
func ListNvmes() (nvmes []ealconfig.PCIAddress, e error) {
	var result listNvmesResult
	ctx := cptr.CtxPut(&result)
	defer cptr.CtxClear(ctx)
	res := eal.CallMain(func() int {
		res := C.spdk_nvme_probe(nil, ctx, C.spdk_nvme_probe_cb(unsafe.Pointer(C.go_nvmeProbed)), nil, nil)
		return int(res)
	}).(int)
	if res != 0 {
		return nil, errors.New("spdk_nvme_probe error")
	}
	return result.nvmes, nil
}

//export go_nvmeProbed
func go_nvmeProbed(ctx unsafe.Pointer, trid *C.struct_spdk_nvme_transport_id, opts *C.struct_spdk_nvme_ctrlr_opts) C.bool {
	pciAddr := ealconfig.MustParsePCIAddress(C.GoString(&trid.traddr[0]))
	result := cptr.CtxGet(ctx).(*listNvmesResult)
	result.nvmes = append(result.nvmes, pciAddr)
	return C.bool(false)
}

// Nvme represents block devices on an NVMe drives.
type Nvme struct {
	// Namespaces is a list of NVMe namespaces as block devices.
	Namespaces []*Info

	pciAddr ealconfig.PCIAddress
}

func (nvme *Nvme) getName() string {
	return fmt.Sprintf("nvme%s", nvme.pciAddr.String())
}

// AttachNvme attaches block devices on an NVMe drives.
func AttachNvme(pciAddr ealconfig.PCIAddress) (nvme *Nvme, e error) {
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
