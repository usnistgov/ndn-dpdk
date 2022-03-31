package eal

/*
#include "../../csrc/core/common.h"
#include <rte_bus_vdev.h>
*/
import "C"
import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/pciaddr"
	"go.uber.org/zap"
)

// JoinDevArgs converts device argument key-value pairs to a string.
// nil values are skipped.
// As a special case, if the map has a "" key, its value would override all other arguments.
func JoinDevArgs(m map[string]any) string {
	var b strings.Builder
	if s, ok := m[""]; ok {
		fmt.Fprint(&b, s)
	} else {
		delim := ""
		for k, v := range m {
			if v == nil {
				continue
			}
			fmt.Fprintf(&b, "%s%s=%v", delim, k, v)
			delim = ","
		}
	}
	return b.String()
}

// ProbePCI requests to probe a PCI device.
func ProbePCI(addr pciaddr.PCIAddress, args map[string]any) error {
	arg := JoinDevArgs(args)
	devargs := addr.String()
	if arg != "" {
		devargs += "," + arg
	}
	devargsC := C.CString(devargs)
	defer C.free(unsafe.Pointer(devargsC))

	logEntry := logger.With(
		zap.String("addr", addr.String()),
		zap.String("args", arg),
	)
	if res := C.rte_dev_probe(devargsC); res != 0 {
		e := MakeErrno(res)
		logEntry.Error("rte_dev_probe error", zap.Error(e))
		return e
	}

	logEntry.Info("PCI device probed")
	return nil
}

// VDev represents a DPDK virtual device.
type VDev struct {
	name   string
	socket NumaSocket
}

// Name returns the device name.
func (vdev VDev) Name() string {
	return vdev.name
}

// NumaSocket returns the NUMA socket of this device, if known.
func (vdev VDev) NumaSocket() NumaSocket {
	return vdev.socket
}

// Close destroys the virtual device.
func (vdev *VDev) Close() error {
	nameC := C.CString(vdev.name)
	defer C.free(unsafe.Pointer(nameC))

	logEntry := logger.With(zap.String("name", vdev.name))
	if res := C.rte_vdev_uninit(nameC); res != 0 {
		e := MakeErrno(res)
		logEntry.Error("rte_vdev_uninit error", zap.Error(e))
		return e
	}

	logEntry.Info("vdev uninitialized")
	return nil
}

// NewVDev creates a virtual device.
func NewVDev(name string, args map[string]any, socket NumaSocket) (vdev *VDev, e error) {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	arg := JoinDevArgs(args)
	argC := C.CString(arg)
	defer C.free(unsafe.Pointer(argC))

	logEntry := logger.With(
		zap.String("name", name),
		zap.String("args", arg),
		socket.ZapField("socket"),
	)
	if res := C.rte_vdev_init(nameC, argC); res != 0 {
		e := MakeErrno(res)
		logEntry.Error("rte_vdev_init error", zap.Error(e))
		return nil, e
	}

	logEntry.Info("vdev initialized")
	return &VDev{
		name:   name,
		socket: socket,
	}, nil
}
