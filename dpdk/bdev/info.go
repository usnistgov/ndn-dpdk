package bdev

/*
#include "../../csrc/dpdk/bdev.h"
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

func (bdi *Info) ptr() *C.struct_spdk_bdev {
	return (*C.struct_spdk_bdev)(bdi)
}

// Name returns device name.
func (bdi *Info) Name() string {
	return C.GoString(C.spdk_bdev_get_name(bdi.ptr()))
}

// ProduceName returns product name.
func (bdi *Info) ProduceName() string {
	return C.GoString(C.spdk_bdev_get_product_name(bdi.ptr()))
}

// BlockSize returns logical block size.
func (bdi *Info) BlockSize() int {
	return int(C.spdk_bdev_get_block_size(bdi.ptr()))
}

// CountBlocks returns size of block device in logical blocks.
func (bdi *Info) CountBlocks() int {
	return int(C.spdk_bdev_get_num_blocks(bdi.ptr()))
}

// IsNvme determines whether the block device is an NVMe device.
func (bdi *Info) IsNvme() bool {
	return bool(C.spdk_bdev_io_type_supported(bdi.ptr(), C.SPDK_BDEV_IO_TYPE_NVME_ADMIN))
}

// Device interface allows retrieving bdev Info.
type Device interface {
	DevInfo() *Info
}

// DevInfo implements Device interface.
func (bdi *Info) DevInfo() *Info {
	return bdi
}
