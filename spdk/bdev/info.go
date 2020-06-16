package bdev

/*
#include "../../csrc/spdk/bdev.h"
*/
import "C"
import (
	"unsafe"
)

// Info provides information about a block device.
type Info C.struct_spdk_bdev

// List returns a list of existing block devices.
func List() (list []*Info) {
	initBdevLib()
	for d := C.spdk_bdev_first(); d != nil; d = C.spdk_bdev_next(d) {
		list = append(list, (*Info)(d))
	}
	return list
}

// Find finds a block device by name.
func Find(name string) *Info {
	initBdevLib()
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	d := C.spdk_bdev_get_by_name(nameC)
	return (*Info)(d)
}

func (bdi *Info) getPtr() *C.struct_spdk_bdev {
	return (*C.struct_spdk_bdev)(bdi)
}

// GetName returns device name.
func (bdi *Info) GetName() string {
	return C.GoString(C.spdk_bdev_get_name(bdi.getPtr()))
}

// GetProductName returns product name.
func (bdi *Info) GetProductName() string {
	return C.GoString(C.spdk_bdev_get_product_name(bdi.getPtr()))
}

// GetBlockSize returns logical block size.
func (bdi *Info) GetBlockSize() int {
	return int(C.spdk_bdev_get_block_size(bdi.getPtr()))
}

// CountBlocks returns size of block device in logical blocks.
func (bdi *Info) CountBlocks() int {
	return int(C.spdk_bdev_get_num_blocks(bdi.getPtr()))
}

// IsNvme determins whether the block device is an NVMe device.
func (bdi *Info) IsNvme() bool {
	return bool(C.spdk_bdev_io_type_supported(bdi.getPtr(), C.SPDK_BDEV_IO_TYPE_NVME_ADMIN))
}

// Device interface allows retrieving bdev Info.
type Device interface {
	GetInfo() *Info
}

// GetInfo implements Device interface.
func (bdi *Info) GetInfo() *Info {
	return bdi
}
