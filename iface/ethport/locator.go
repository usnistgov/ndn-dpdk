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

// UDPPortGTP is the standard UDP port for GTP-U.
const UDPPortGTP = C.RTE_GTPU_UDP_PORT

// SchemePassthru indicates a pass-through face.
const SchemePassthru = "passthru"

func (loc *LocatorC) ptr() *C.EthLocator {
	return (*C.EthLocator)(unsafe.Pointer(loc))
}

func (loc LocatorC) toXDP() []byte {
	var buf [C.sizeof_EthXdpLocator]byte
	C.EthXdpLocator_Prepare((*C.EthXdpLocator)(unsafe.Pointer(&buf)), loc.ptr())
	return buf[:]
}

// Locator is an Ethernet-based face locator.
type Locator interface {
	iface.Locator
	EthLocatorC() LocatorC
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
	aC, bC := a.EthLocatorC(), b.EthLocatorC()
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
	locC := loc.EthLocatorC()
	C.EthRxMatch_Prepare((*C.EthRxMatch)(&match), locC.ptr())
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
//
//	newBurst: whether pkt is the first packet in a burst. It increments UDP source port in VXLAN
//	          headers. If NDN network layer packet is fragmented, only the first fragment might
//	          start a new burst, so that all fragments have the same UDP source port.
func (hdr TxHdr) Prepend(pkt *pktmbuf.Packet, newBurst bool) {
	C.EthTxHdr_Prepend((*C.EthTxHdr)(&hdr), (*C.struct_rte_mbuf)(pkt.Ptr()), C.bool(newBurst))
}

// NewTxHdr creates TxHdr from a locator.
func NewTxHdr(loc Locator, hasChecksumOffloads bool) (hdr TxHdr) {
	locC := loc.EthLocatorC()
	C.EthTxHdr_Prepare((*C.EthTxHdr)(&hdr), locC.ptr(), C.bool(hasChecksumOffloads))
	return
}
