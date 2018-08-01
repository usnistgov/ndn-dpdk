package dpdk

/*
#include <rte_config.h>
#include <rte_bus_vdev.h>
#include <stdlib.h>
*/
import "C"
import (
	"unsafe"
)

func CreateVdev(name, args string) error {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	argsC := C.CString(args)
	defer C.free(unsafe.Pointer(argsC))

	if res := C.rte_vdev_init(nameC, argsC); res != 0 {
		return GetErrno()
	}
	return nil
}

func DestroyVdev(name string) error {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))

	if res := C.rte_vdev_uninit(nameC); res != 0 {
		return GetErrno()
	}
	return nil
}
