package fibreplica

/*
#include "../../../csrc/fib/entry.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// Entry represents a FIB entry.
type Entry C.FibEntry

func entryFromPtr(c *C.FibEntry) *Entry {
	return (*Entry)(c)
}

func (entry *Entry) ptr() *C.FibEntry {
	return (*C.FibEntry)(entry)
}

// Ptr returns *C.FibEntry pointer.
func (entry *Entry) Ptr() unsafe.Pointer {
	return unsafe.Pointer(entry)
}

// Read converts Entry to fibdef.Entry.
func (entry *Entry) Read() (de fibdef.Entry) {
	c := entry.ptr()
	if c.height > 0 {
		panic("cannot Read virtual entry")
	}

	de.Name.UnmarshalBinary(cptr.AsByteSlice(c.nameV[:c.nameL]))

	de.Nexthops = make([]iface.ID, int(c.nNexthops))
	for i := range de.Nexthops {
		de.Nexthops[i] = iface.ID(c.nexthops[i])
	}

	ptrStrategy := C.FibEntry_PtrStrategy(c)
	de.Strategy = strategycode.FromPtr(unsafe.Pointer(*ptrStrategy)).ID()
	return
}

// AccCounters adds to counters.
func (entry *Entry) AccCounters(cnt *fibdef.EntryCounters, t *Table) {
	c := entry.Real().ptr()
	for i := 0; i < t.nDyns; i++ {
		dyn := C.FibEntry_PtrDyn(c, C.int(i))
		cnt.NRxInterests += uint64(dyn.nRxInterests)
		cnt.NRxData += uint64(dyn.nRxData)
		cnt.NRxNacks += uint64(dyn.nRxNacks)
		cnt.NTxInterests += uint64(dyn.nTxInterests)
	}
}

// IsVirt determines whether this is a virtual entry.
func (entry *Entry) IsVirt() bool {
	return entry.height > 0
}

// Real returns the real entry linked from this entry.
func (entry *Entry) Real() *Entry {
	return entryFromPtr(C.FibEntry_GetReal(entry.ptr()))
}

// FibSeqNum returns the FIB insertion sequence number recorded in this entry.
func (entry *Entry) FibSeqNum() uint32 {
	return uint32(entry.seqNum)
}

func (entry *Entry) assignReal(u *fibdef.RealUpdate) {
	c := entry.ptr()
	c.height = 0

	nameV, _ := u.Name.MarshalBinary()
	c.nameL = C.uint16_t(copy(cptr.AsByteSlice(&c.nameV), nameV))
	c.nComps = C.uint8_t(len(u.Name))

	c.nNexthops = C.uint8_t(len(u.Nexthops))
	for i, nh := range u.Nexthops {
		c.nexthops[i] = C.FaceID(nh)
	}

	ptrStrategy := C.FibEntry_PtrStrategy(c)
	*ptrStrategy = (*C.StrategyCode)(strategycode.Get(u.Strategy).Ptr())
}

func (entry *Entry) assignVirt(u *fibdef.VirtUpdate, real *Entry) {
	c := entry.ptr()
	c.height = C.uint8_t(u.Height)

	nameV, _ := u.Name.MarshalBinary()
	c.nameL = C.uint16_t(copy(cptr.AsByteSlice(&c.nameV), nameV))
	c.nComps = C.uint8_t(len(u.Name))

	ptrReal := C.FibEntry_PtrRealEntry(c)
	*ptrReal = real.ptr()
}
