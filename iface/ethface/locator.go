package ethface

/*
#include "../../csrc/ethface/locator.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

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

type ethLocator interface {
	iface.Locator

	// cLoc converts to C.EthLocator.
	cLoc() cLocator

	// maxRxQueues returns maximum number of RX queues when using RxFlow.
	maxRxQueues() int
}

// LocatorCanCoexist determines whether two locators can coexist on the same port.
func LocatorCanCoexist(a, b iface.Locator) bool {
	return a.(ethLocator).cLoc().canCoexist(b.(ethLocator).cLoc())
}
