package ethface

/*
#include "../../csrc/ethface/locator.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// LocatorConflictError indicates that the locator of a new face conflicts with an existing face.
type LocatorConflictError struct {
	a, b ethLocator
}

func (e LocatorConflictError) Error() string {
	return fmt.Sprintf("locator %s conflicts with %s", iface.LocatorString(e.a), iface.LocatorString(e.b))
}

func (loc *cLocator) ptr() *C.EthLocator {
	return (*C.EthLocator)(unsafe.Pointer(loc))
}

func (loc cLocator) canCoexist(other cLocator) bool {
	return bool(C.EthLocator_CanCoexist(loc.ptr(), other.ptr()))
}

func (loc cLocator) sizeofHeader() int {
	var hdr C.EthTxHdr
	C.EthTxHdr_Prepare(&hdr, loc.ptr(), true)
	return int(hdr.len - hdr.l2len)
}

// FaceConfig contains additional face configuration.
// They appear as input-only fields of EtherLocator.
type FaceConfig struct {
	iface.Config

	// EthDev causes the face to be created on a specific Ethernet adapter.
	// This allows setting a local MAC address that differs from the physical MAC address.
	//
	// If omitted, local MAC address is used to find the Ethernet adapter.
	//
	// In either case, a Port must be created on the Ethernet adapter before creating faces.
	EthDev ethdev.EthDev `json:"-"`

	// Port is GraphQL ID of the EthDev.
	// This field has the same semantics as EthDev.
	// If both EthDev and Port are specified, EthDev takes priority.
	Port string `json:"port,omitempty"`

	// MaxRxQueues is the maximum number of RX queues for this face.
	// It is meaningful only if the port is using RxFlow.
	// For most DPDK drivers, it is effective in improving performance on VXLAN face only.
	//
	// Default is 1.
	MaxRxQueues int `json:"maxRxQueues,omitempty"`

	// DisableTxMultiSegOffload forces every packet to be copied into a linear buffer in software.
	DisableTxMultiSegOffload bool `json:"disableTxMultiSegOffload,omitempty"`

	// DisableTxChecksumOffload disables the usage of IPv4 and UDP checksum offloads.
	DisableTxChecksumOffload bool `json:"disableTxChecksumOffload,omitempty"`

	// privFaceConfig is hidden from JSON output.
	privFaceConfig *FaceConfig
}

func (cfg FaceConfig) faceConfig() FaceConfig {
	if cfg.privFaceConfig != nil {
		return *cfg.privFaceConfig
	}
	return cfg
}

func (cfg FaceConfig) hideFaceConfigFromJSON() FaceConfig {
	return FaceConfig{privFaceConfig: &cfg}
}

type ethLocator interface {
	iface.Locator

	// cLoc converts to C.EthLocator.
	cLoc() cLocator

	faceConfig() FaceConfig
}

// LocatorCanCoexist determines whether two locators can coexist on the same port.
func LocatorCanCoexist(a, b iface.Locator) bool {
	return a.(ethLocator).cLoc().canCoexist(b.(ethLocator).cLoc())
}

// RxMatchFunc matches an incoming packet against the locator and strips headers.
type RxMatchFunc func(m *pktmbuf.Packet) bool

// LocatorRxMatch creates RxMatchFunc from a locator.
func LocatorRxMatch(loc iface.Locator) RxMatchFunc {
	cLoc := loc.(ethLocator).cLoc()
	var match C.EthRxMatch
	C.EthRxMatch_Prepare(&match, cLoc.ptr())
	return func(m *pktmbuf.Packet) bool {
		return bool(C.EthRxMatch_Match(&match, (*C.struct_rte_mbuf)(m.Ptr())))
	}
}

// TxHdrFunc prepends headers to an outgoing packet according to the locator
type TxHdrFunc func(m *pktmbuf.Packet, newBurst bool)

// LocatorTxHdr creates TxHdrFunc from a locator.
func LocatorTxHdr(loc iface.Locator, hasChecksumOffloads bool) TxHdrFunc {
	cLoc := loc.(ethLocator).cLoc()
	var hdr C.EthTxHdr
	C.EthTxHdr_Prepare(&hdr, cLoc.ptr(), C.bool(hasChecksumOffloads))
	return func(m *pktmbuf.Packet, newBurst bool) {
		C.EthTxHdr_Prepend(&hdr, (*C.struct_rte_mbuf)(m.Ptr()), C.bool(newBurst))
	}
}
