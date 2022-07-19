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

// UDPPortVXLAN is the default UDP destination port for VXLAN.
const UDPPortVXLAN = C.RTE_VXLAN_DEFAULT_PORT

func (loc *CLocator) ptr() *C.EthLocator {
	return (*C.EthLocator)(unsafe.Pointer(loc))
}

func (loc CLocator) toXDP() (b []byte) {
	b = make([]byte, C.sizeof_EthXdpLocator)
	C.EthXdpLocator_Prepare((*C.EthXdpLocator)(unsafe.Pointer(&b[0])), loc.ptr())
	return
}

// Locator is an Ethernet-based face locator.
type Locator interface {
	iface.Locator
	EthCLocator() CLocator
	EthFaceConfig() FaceConfig
}

// LocatorConflictError indicates that the locator of a new face conflicts with an existing face.
type LocatorConflictError struct {
	a, b Locator
}

func (e LocatorConflictError) Error() string {
	return fmt.Sprintf("locator %s conflicts with %s", iface.LocatorString(e.a), iface.LocatorString(e.b))
}

// CheckLocatorCoexist determines whether two locators can coexist on the same port.
func CheckLocatorCoexist(a, b Locator) error {
	aC, bC := a.EthCLocator(), b.EthCLocator()
	if C.EthLocator_CanCoexist(aC.ptr(), bC.ptr()) {
		return nil
	}
	return LocatorConflictError{a: a, b: b}
}

// RxMatch contains prepared buffer to match incoming packets with a locator.
type RxMatch C.EthRxMatch

func (match RxMatch) copyToC(c *C.EthRxMatch) {
	*c = *(*C.EthRxMatch)(&match)
}

// Match determines whether an incoming packet matches the locator.
func (match RxMatch) Match(pkt *pktmbuf.Packet) bool {
	return bool(C.EthRxMatch_Match((*C.EthRxMatch)(&match), (*C.struct_rte_mbuf)(pkt.Ptr())))
}

// HdrLen returns header length.
func (match RxMatch) HdrLen() int {
	return int(match.len)
}

// NewRxMatch creates RxMatch from a locator.
func NewRxMatch(loc Locator) (match RxMatch) {
	cLoc := loc.EthCLocator()
	C.EthRxMatch_Prepare((*C.EthRxMatch)(&match), cLoc.ptr())
	return
}

// TxHdr contains prepare buffer to prepend headers to outgoing packets.
type TxHdr C.EthTxHdr

func (hdr TxHdr) copyToC(c *C.EthTxHdr) {
	*c = *(*C.EthTxHdr)(&hdr)
}

// IPLen returns the total length of IP, UDP, and VXLAN headers.
func (hdr TxHdr) IPLen() int {
	return int(hdr.len - hdr.l2len)
}

// Prepend prepends headers to an outgoing packet.
//  newBurst: whether pkt is the first packet in a burst. It increments UDP source port in VXLAN
//            headers. If NDN network layer packet is fragmented, only the first fragment might
//            start a new burst, so that all fragments have the same UDP source port.
func (hdr TxHdr) Prepend(pkt *pktmbuf.Packet, newBurst bool) {
	C.EthTxHdr_Prepend((*C.EthTxHdr)(&hdr), (*C.struct_rte_mbuf)(pkt.Ptr()), C.bool(newBurst))
}

// NewTxHdr creates TxHdr from a locator.
func NewTxHdr(loc Locator, hasChecksumOffloads bool) (hdr TxHdr) {
	cLoc := loc.EthCLocator()
	C.EthTxHdr_Prepare((*C.EthTxHdr)(&hdr), cLoc.ptr(), C.bool(hasChecksumOffloads))
	return
}
