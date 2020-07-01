package fib

/*
#include "../../csrc/fib/entry.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

const (
	// MaxNameLength is the maximum TLV-LENGTH of a FIB entry name.
	MaxNameLength = C.FIB_ENTRY_MAX_NAME_LEN

	// MaxNexthops is the maximum number of nexthops in a FIB entry.
	MaxNexthops = C.FIB_ENTRY_MAX_NEXTHOPS
)

// Entry represents a FIB entry.
type Entry CEntry

// Name returns the entry name.
func (entry *Entry) Name() (name ndn.Name) {
	c := (*CEntry)(entry)
	name.UnmarshalBinary(c.NameV[:c.NameL])
	return name
}

// SetName sets the entry name.
func (entry *Entry) SetName(name ndn.Name) error {
	nameV, _ := name.MarshalBinary()
	nameL := len(nameV)
	if nameL > MaxNameLength {
		return fmt.Errorf("FIB entry name cannot exceed %d octets", MaxNameLength)
	}

	c := (*CEntry)(entry)
	c.NameL = uint16(copy(c.NameV[:], nameV))
	c.NComps = uint8(len(name))
	return nil
}

// GetNexthops returns a list of nexthops.
func (entry *Entry) GetNexthops() (nexthops []iface.ID) {
	c := (*CEntry)(entry)
	nexthops = make([]iface.ID, int(c.NNexthops))
	for i := range nexthops {
		nexthops[i] = iface.ID(c.Nexthops[i])
	}
	return nexthops
}

// SetNexthops sets a list of nexthops.
func (entry *Entry) SetNexthops(nexthops []iface.ID) error {
	count := len(nexthops)
	if count > MaxNexthops {
		return fmt.Errorf("FIB entry cannot have more than %d nexthops", MaxNexthops)
	}

	c := (*CEntry)(entry)
	c.NNexthops = uint8(count)
	for i, nh := range nexthops {
		c.Nexthops[i] = uint16(nh)
	}
	return nil
}

// GetStrategy returns the forwarding strategy.
func (entry *Entry) GetStrategy() strategycode.StrategyCode {
	c := (*CEntry)(entry)
	return strategycode.FromPtr(unsafe.Pointer(c.Union_strategy_realEntry))
}

// SetStrategy sets the forwarding strategy.
func (entry *Entry) SetStrategy(sc strategycode.StrategyCode) {
	c := (*CEntry)(entry)
	c.Union_strategy_realEntry = (*byte)(sc.Ptr())
}

// GetSeqNum returns FIB insertion sequence number.
// This number is automatically assigned when a FIB entry is inserted/overwritten.
// Its change signifies that the FIB entry has been updated.
// This function is only available on a retrieved FIB entry.
func (entry *Entry) GetSeqNum() uint32 {
	c := (*CEntry)(entry)
	return c.SeqNum
}

func entryFromPtr(c *C.FibEntry) *Entry {
	return (*Entry)(unsafe.Pointer(c))
}

func (entry *Entry) ptr() *C.FibEntry {
	return (*C.FibEntry)(unsafe.Pointer(entry))
}

func (entry *Entry) copyFrom(src *Entry) {
	C.FibEntry_Copy(entry.ptr(), src.ptr())
}

func (entry *Entry) isVirt() bool {
	return entry != nil && (*CEntry)(entry).MaxDepth > 0
}

func (entry *Entry) getReal() *Entry {
	return entryFromPtr(C.FibEntry_GetReal(entry.ptr()))
}

func (entry *Entry) setMaxDepthReal(maxDepth int, real *Entry) {
	c := (*CEntry)(entry)
	c.MaxDepth = uint8(maxDepth)
	c.Union_strategy_realEntry = (*byte)(unsafe.Pointer(real))
}
