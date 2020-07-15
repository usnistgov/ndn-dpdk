package fib

/*
#include "../../csrc/fib/entry.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

const (
	// MaxNameLength is the maximum TLV-LENGTH of a FIB entry name.
	MaxNameLength = C.FibMaxNameLength

	// MaxNexthops is the maximum number of nexthops in a FIB entry.
	MaxNexthops = C.FibMaxNexthops
)

// Entry represents a FIB entry.
type Entry C.FibEntry

func entryFromPtr(c *C.FibEntry) *Entry {
	return (*Entry)(c)
}

func (entry *Entry) ptr() *C.FibEntry {
	return (*C.FibEntry)(entry)
}

// Name returns the entry name.
func (entry *Entry) Name() (name ndn.Name) {
	c := entry.ptr()
	name.UnmarshalBinary(cptr.AsByteSlice(c.nameV[:c.nameL]))
	return name
}

// SetName sets the entry name.
func (entry *Entry) SetName(name ndn.Name) error {
	nameV, _ := name.MarshalBinary()
	nameL := len(nameV)
	if nameL > MaxNameLength {
		return fmt.Errorf("FIB entry name cannot exceed %d octets", MaxNameLength)
	}

	c := entry.ptr()
	c.nameL = C.uint16_t(copy(cptr.AsByteSlice(&c.nameV), nameV))
	c.nComps = C.uint8_t(len(name))
	return nil
}

// Nexthops returns a list of nexthops.
func (entry *Entry) Nexthops() (nexthops []iface.ID) {
	c := entry.ptr()
	nexthops = make([]iface.ID, int(c.nNexthops))
	for i := range nexthops {
		nexthops[i] = iface.ID(c.nexthops[i])
	}
	return nexthops
}

// SetNexthops sets a list of nexthops.
func (entry *Entry) SetNexthops(nexthops []iface.ID) error {
	count := len(nexthops)
	if count > MaxNexthops {
		return fmt.Errorf("FIB entry cannot have more than %d nexthops", MaxNexthops)
	}

	c := entry.ptr()
	c.nNexthops = C.uint8_t(count)
	for i, nh := range nexthops {
		c.nexthops[i] = C.FaceID(nh)
	}
	return nil
}

// Strategy returns the forwarding strategy.
func (entry *Entry) Strategy() strategycode.StrategyCode {
	ptrStrategy := C.FibEntry_PtrStrategy(entry.ptr())
	return strategycode.FromPtr(unsafe.Pointer(*ptrStrategy))
}

// SetStrategy sets the forwarding strategy.
func (entry *Entry) SetStrategy(sc strategycode.StrategyCode) {
	ptrStrategy := C.FibEntry_PtrStrategy(entry.ptr())
	*ptrStrategy = (*C.StrategyCode)(sc.Ptr())
}

// FibSeqNum returns FIB insertion sequence number.
// This number is automatically assigned when a FIB entry is inserted/overwritten.
// Its change signifies that the FIB entry has been updated.
// This function is only available on a retrieved FIB entry.
func (entry *Entry) FibSeqNum() uint32 {
	return uint32(entry.ptr().seqNum)
}

func (entry *Entry) copyFrom(src *Entry) {
	C.FibEntry_Copy(entry.ptr(), src.ptr())
}

func (entry *Entry) isVirt() bool {
	return entry != nil && entry.ptr().maxDepth > 0
}

func (entry *Entry) getReal() *Entry {
	return entryFromPtr(C.FibEntry_GetReal(entry.ptr()))
}

func (entry *Entry) setMaxDepthReal(maxDepth int, real *Entry) {
	c := entry.ptr()
	c.maxDepth = C.uint8_t(maxDepth)
	ptrReal := C.FibEntry_PtrRealEntry(c)
	*ptrReal = real.ptr()
}
