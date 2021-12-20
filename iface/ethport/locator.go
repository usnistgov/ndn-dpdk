package ethport

/*
#include "../../csrc/ethface/locator.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
)

func (loc *CLocator) ptr() *C.EthLocator {
	return (*C.EthLocator)(unsafe.Pointer(loc))
}

func (loc CLocator) canCoexist(other CLocator) bool {
	return bool(C.EthLocator_CanCoexist(loc.ptr(), other.ptr()))
}

func (loc CLocator) sizeofHeader() int {
	var hdr C.EthTxHdr
	C.EthTxHdr_Prepare(&hdr, loc.ptr(), true)
	return int(hdr.len - hdr.l2len)
}

// LocatorConflictError indicates that the locator of a new face conflicts with an existing face.
type LocatorConflictError struct {
	a, b Locator
}

func (e LocatorConflictError) Error() string {
	return fmt.Sprintf("locator %s conflicts with %s", iface.LocatorString(e.a), iface.LocatorString(e.b))
}

// Locator is an Ethernet-based face locator.
type Locator interface {
	iface.Locator
	EthCLocator() CLocator
	EthFaceConfig() FaceConfig
}

// LocatorCanCoexist determines whether two locators can coexist on the same port.
func LocatorCanCoexist(a, b Locator) bool {
	return a.EthCLocator().canCoexist(b.EthCLocator())
}

// RxMatchFunc matches an incoming packet against the locator.
// Headers are stripped after a successful match.
type RxMatchFunc func(m *pktmbuf.Packet) bool

// LocatorRxMatch creates RxMatchFunc from a locator.
func LocatorRxMatch(loc Locator) RxMatchFunc {
	cLoc := loc.EthCLocator()
	var match C.EthRxMatch
	C.EthRxMatch_Prepare(&match, cLoc.ptr())
	return func(m *pktmbuf.Packet) bool {
		return bool(C.EthRxMatch_Match(&match, (*C.struct_rte_mbuf)(m.Ptr())))
	}
}

// TxHdrFunc prepends headers to an outgoing packet according to the locator.
type TxHdrFunc func(m *pktmbuf.Packet, newBurst bool)

// LocatorTxHdr creates TxHdrFunc from a locator.
func LocatorTxHdr(loc Locator, hasChecksumOffloads bool) TxHdrFunc {
	cLoc := loc.EthCLocator()
	var hdr C.EthTxHdr
	C.EthTxHdr_Prepare(&hdr, cLoc.ptr(), C.bool(hasChecksumOffloads))
	return func(m *pktmbuf.Packet, newBurst bool) {
		C.EthTxHdr_Prepend(&hdr, (*C.struct_rte_mbuf)(m.Ptr()), C.bool(newBurst))
	}
}
