package spdk

/*
#include <spdk/bdev.h>

extern void go_bdevInitialized(void* ctx, int rc);
extern void go_bdevRemove(void* ctx);
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"

	"ndn-dpdk/dpdk"
)

var initBdevLibOnce sync.Once

// Initialize SPDK block device library.
func InitBdevLib() {
	initBdevLibOnce.Do(func() {
		MainThread.Call(func() { C.spdk_bdev_initialize(C.spdk_bdev_init_cb(C.go_bdevInitialized), nil) })
	})
}

//export go_bdevInitialized
func go_bdevInitialized(ctx unsafe.Pointer, rc C.int) {
	if rc != 0 {
		panic(fmt.Sprintf("spdk_bdev_initialize error %v", dpdk.Errno(rc)))
	}
}

// Information about a block device.
type BdevInfo struct {
	c *C.struct_spdk_bdev
}

// List existing block devices.
func ListBdevs() (list []BdevInfo) {
	for d := C.spdk_bdev_first(); d != nil; d = C.spdk_bdev_next(d) {
		list = append(list, BdevInfo{d})
	}
	return list
}

// Find block device by name.
func FindBdev(name string) (bdi BdevInfo, ok bool) {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	d := C.spdk_bdev_get_by_name(nameC)
	if d == nil {
		return BdevInfo{}, false
	}
	return BdevInfo{d}, true
}

func mustFindBdev(name string) BdevInfo {
	bdi, ok := FindBdev(name)
	if !ok {
		panic(fmt.Sprintf("bdev %s not found", name))
	}
	return bdi
}

func (bdi BdevInfo) GetName() string {
	return C.GoString(C.spdk_bdev_get_name(bdi.c))
}

func (bdi BdevInfo) GetProductName() string {
	return C.GoString(C.spdk_bdev_get_product_name(bdi.c))
}

func (bdi BdevInfo) GetBlockSize() int {
	return int(C.spdk_bdev_get_block_size(bdi.c))
}

func (bdi BdevInfo) CountBlocks() int {
	return int(C.spdk_bdev_get_num_blocks(bdi.c))
}
