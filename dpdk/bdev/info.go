package bdev

/*
#include "../../csrc/dpdk/bdev.h"
#include <spdk/bdev_module.h>
*/
import "C"
import (
	"io"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// IOType represents an I/O type.
type IOType int

const (
	IORead      IOType = C.SPDK_BDEV_IO_TYPE_READ
	IOWrite     IOType = C.SPDK_BDEV_IO_TYPE_WRITE
	IOUnmap     IOType = C.SPDK_BDEV_IO_TYPE_UNMAP
	IONvmeAdmin IOType = C.SPDK_BDEV_IO_TYPE_NVME_ADMIN
	IONvmeIO    IOType = C.SPDK_BDEV_IO_TYPE_NVME_IO
)

// Device interface represents a device.
type Device interface {
	DevInfo() *Info
}

// DeviceCloser interface is a device that can be closed.
type DeviceCloser interface {
	Device
	io.Closer
}

// Info provides information about a block device.
type Info C.struct_spdk_bdev

var _ zapcore.ObjectMarshaler = (*Info)(nil)

func (bdi *Info) ptr() *C.struct_spdk_bdev {
	return (*C.struct_spdk_bdev)(bdi)
}

// Name returns device name.
func (bdi *Info) Name() string {
	return C.GoString(C.spdk_bdev_get_name(bdi.ptr()))
}

// ProductName returns product name.
func (bdi *Info) ProductName() string {
	return C.GoString(C.spdk_bdev_get_product_name(bdi.ptr()))
}

// BlockSize returns logical block size in octets.
func (bdi *Info) BlockSize() int {
	return int(C.spdk_bdev_get_block_size(bdi.ptr()))
}

// CountBlocks returns size of block device in logical blocks.
func (bdi *Info) CountBlocks() int64 {
	return int64(C.spdk_bdev_get_num_blocks(bdi.ptr()))
}

// WriteUnitSize returns write unit size in logical blocks.
func (bdi *Info) WriteUnitSize() int {
	return int(C.spdk_bdev_get_write_unit_size(bdi.ptr()))
}

// BufAlign returns minimum I/O buffer address alignment in octets.
func (bdi *Info) BufAlign() int {
	return int(C.spdk_bdev_get_buf_align(bdi.ptr()))
}

// OptimalIOBoundary returns optimal I/O boundary in logical blocks and whether it's mandatory.
func (bdi *Info) OptimalIOBoundary() (boundary int, mandatory bool) {
	return int(C.spdk_bdev_get_optimal_io_boundary(bdi.ptr())), bool(bdi.split_on_optimal_io_boundary)
}

// HasWriteCache returns whether write cache is enabled.
func (bdi *Info) HasWriteCache() bool {
	return bool(C.spdk_bdev_has_write_cache(bdi.ptr()))
}

// HasIOType determines whether the I/O type is supported.
func (bdi *Info) HasIOType(ioType IOType) bool {
	return bool(C.spdk_bdev_io_type_supported(bdi.ptr(), C.enum_spdk_bdev_io_type(ioType)))
}

// DriverInfo returns driver-specific information.
func (bdi *Info) DriverInfo() (value any) {
	var res C.int
	e := spdkenv.CaptureJSON(spdkenv.JSONObject(func(w unsafe.Pointer) {
		res = C.spdk_bdev_dump_info_json(bdi.ptr(), (*C.struct_spdk_json_write_ctx)(w))
	}), &value)
	if res != 0 || e != nil {
		logger.Warn("spdk_bdev_dump_info_json error",
			zap.Int("res", int(res)),
			zap.Error(e),
		)
		return nil
	}
	return value
}

// DevInfo implements Device interface.
func (bdi *Info) DevInfo() *Info {
	return bdi
}

// MarshalLogObject implements zapcore.ObjectMarshaler interface.
func (bdi *Info) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("name", bdi.Name())
	enc.AddString("product-name", bdi.ProductName())
	enc.AddInt("block-size", bdi.BlockSize())
	enc.AddInt64("block-count", bdi.CountBlocks())
	enc.AddInt("write-unit-size", bdi.WriteUnitSize())
	enc.AddInt("buf-align", bdi.BufAlign())
	boundary, mandatory := bdi.OptimalIOBoundary()
	enc.AddInt("optimal-io-boundary", boundary)
	enc.AddBool("optimal-io-boundary-mandatory", mandatory)
	enc.AddBool("has-write-cache", bdi.HasWriteCache())
	enc.AddBool("can-read", bdi.HasIOType(IORead))
	enc.AddBool("can-write", bdi.HasIOType(IOWrite))
	enc.AddBool("can-unmap", bdi.HasIOType(IOUnmap))
	enc.AddReflected("driver-info", bdi.DriverInfo())
	return nil
}

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
	for d := C.spdk_bdev_first(); d != nil; d = C.spdk_bdev_next(d) {
		bdi := (*Info)(d)
		if bdi.Name() == name {
			return bdi
		}
	}
	return nil
	// C.spdk_bdev_get_by_name is only available in SPDK thread
}

func mustFind(name string) *Info {
	info := Find(name)
	if info == nil {
		logger.Panic("bdev not found",
			zap.String("name", name),
		)
	}
	return info
}

func deleteByName(method, name string) error {
	args := struct {
		Name string `json:"name"`
	}{
		Name: name,
	}
	var ok bool
	return spdkenv.RPC(method, args, &ok)
}
