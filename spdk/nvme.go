package spdk

/*
#include "../core/common.h"
#include <spdk/nvme.h>

extern bool go_nvmeProbed(void* ctx, struct spdk_nvme_transport_id* trid, struct spdk_nvme_ctrlr_opts* opts);
*/
import "C"
import (
	"errors"
	"unsafe"

	"ndn-dpdk/dpdk"
)

type listNvmesResult struct {
	nvmes []dpdk.PciAddress
}

func ListNvmes() (nvmes []dpdk.PciAddress, e error) {
	var result listNvmesResult
	ctx := ctxPut(&result)
	res := MainThread.Call(func() int {
		res := C.spdk_nvme_probe(nil, ctx, C.spdk_nvme_probe_cb(unsafe.Pointer(C.go_nvmeProbed)), nil, nil)
		return int(res)
	}).(int)
	ctxClear(ctx)
	if res != 0 {
		return nil, errors.New("spdk_nvme_probe error")
	}
	return result.nvmes, nil
}

//export go_nvmeProbed
func go_nvmeProbed(ctx unsafe.Pointer, trid *C.struct_spdk_nvme_transport_id, opts *C.struct_spdk_nvme_ctrlr_opts) C.bool {
	pciAddr := dpdk.MustParsePciAddress(C.GoString(&trid.traddr[0]))
	result := ctxGet(ctx).(*listNvmesResult)
	result.nvmes = append(result.nvmes, pciAddr)
	return C.bool(false)
}
