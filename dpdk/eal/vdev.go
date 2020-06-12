package eal

/*
#include "../../core/common.h"
#include <rte_bus_vdev.h>
*/
import "C"
import (
	"unsafe"
)

// Create virtual device.
func CreateVdev(name, args string) error {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	argsC := C.CString(args)
	defer C.free(unsafe.Pointer(argsC))

	if res := C.rte_vdev_init(nameC, argsC); res != 0 {
		return Errno(-res)
	}
	return nil
}

// Destroy virtual device.
func DestroyVdev(name string) error {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))

	if res := C.rte_vdev_uninit(nameC); res != 0 {
		return Errno(-res)
	}
	return nil
}
