// Package ethringdev contains bindings of DPDK net_eth_ring driver.
package ethringdev

/*
#include "../../../csrc/core/common.h"
#include <rte_ethdev.h>
#include <rte_eth_ring.h>
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
)

var logger = logging.New("ethringdev")

// New creates an EthDev from a set of software FIFOs.
func New(rxRings, txRings []*ringbuffer.Ring, socket eal.NumaSocket) (dev ethdev.EthDev, e error) {
	nameC := C.CString(eal.AllocObjectID("ethringdev.EthDev"))
	defer C.free(unsafe.Pointer(nameC))

	rxRingPtr, rxRingCount := cptr.ParseCptrArray(rxRings)
	txRingPtr, txRingCount := cptr.ParseCptrArray(txRings)

	res := C.rte_eth_from_rings(nameC,
		(**C.struct_rte_ring)(rxRingPtr), C.uint(rxRingCount),
		(**C.struct_rte_ring)(txRingPtr), C.uint(txRingCount),
		C.uint(socket.ID()))
	if res < 0 {
		return nil, eal.GetErrno()
	}
	dev = ethdev.FromID(int(res))

	mac := macaddr.MakeRandom(false)
	var macC C.struct_rte_ether_addr
	copy(cptr.AsByteSlice(macC.addr_bytes[:]), mac)
	res = C.rte_eth_dev_mac_addr_add(C.uint16_t(dev.ID()), &macC, 0)
	if res != 0 {
		dev.Close()
		return nil, eal.MakeErrno(res)
	}

	return dev, nil
}
