package eal

/*
#include "../../csrc/core/common.h"
#include <rte_bus_vdev.h>
*/
import "C"
import (
	"unsafe"
)

// VDev represents a DPDK virtual device.
type VDev struct {
	name   string
	socket NumaSocket
}

// NewVDev creates a virtual device.
func NewVDev(name, args string, socket NumaSocket) (vdev *VDev, e error) {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	argsC := C.CString(args)
	defer C.free(unsafe.Pointer(argsC))

	if res := C.rte_vdev_init(nameC, argsC); res != 0 {
		return nil, Errno(-res)
	}

	vdev = &VDev{
		name:   name,
		socket: socket,
	}
	return vdev, nil
}

// Name returns the device name.
func (vdev *VDev) Name() string {
	return vdev.name
}

// NumaSocket returns the NUMA socket of this device, if known.
func (vdev *VDev) NumaSocket() NumaSocket {
	return vdev.socket
}

// Close destroys the virtual device.
func (vdev *VDev) Close() error {
	nameC := C.CString(vdev.name)
	defer C.free(unsafe.Pointer(nameC))

	if res := C.rte_vdev_uninit(nameC); res != 0 {
		return Errno(-res)
	}
	return nil
}
