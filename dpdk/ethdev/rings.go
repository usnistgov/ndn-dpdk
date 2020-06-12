package ethdev

/*
#include "ethdev.h"
#include <rte_eth_ring.h>
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/core/cptr"
	"ndn-dpdk/dpdk/eal"
	"ndn-dpdk/dpdk/ringbuffer"
)

// NewFromRings creates an EthDev using net/ring driver.
func NewFromRings(name string, rxRings []*ringbuffer.Ring, txRings []*ringbuffer.Ring, socket eal.NumaSocket) (dev EthDev, e error) {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	rxRingPtr, rxRingCount := cptr.ParseCptrArray(rxRings)
	txRingPtr, txRingCount := cptr.ParseCptrArray(txRings)
	res := C.rte_eth_from_rings(nameC,
		(**C.struct_rte_ring)(rxRingPtr), C.uint(rxRingCount),
		(**C.struct_rte_ring)(txRingPtr), C.uint(txRingCount),
		C.uint(socket.ID()))
	if res < 0 {
		return EthDev{}, eal.GetErrno()
	}
	return FromID(int(res)), nil
}
