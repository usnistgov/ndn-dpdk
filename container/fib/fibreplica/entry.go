package fibreplica

/*
#include "../../../csrc/fib/entry.h"

static_assert(offsetof(FibEntry, strategy) == offsetof(FibEntry, realEntry), "");
enum { c_FibEntry_StrategyRealOffset = offsetof(FibEntry, strategy) };
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

// Ptr returns *C.FibEntry pointer.
func (entry *Entry) Ptr() unsafe.Pointer {
	return unsafe.Pointer(entry)
}

func (entry *Entry) ptr() *C.FibEntry {
	return (*C.FibEntry)(entry)
}

func (entry *Entry) ptrStrategy() **C.StrategyCode {
	return (**C.StrategyCode)(unsafe.Add(entry.Ptr(), C.c_FibEntry_StrategyRealOffset))
}

func (entry *Entry) ptrReal() **C.FibEntry {
	return (**C.FibEntry)(unsafe.Add(entry.Ptr(), C.c_FibEntry_StrategyRealOffset))
}

// Read converts Entry to fibdef.Entry.
func (entry *Entry) Read() (de fibdef.Entry) {
	if entry.height > 0 {
		panic("cannot Read virtual entry")
	}

	de.Name.UnmarshalBinary(cptr.AsByteSlice(entry.nameV[:entry.nameL]))

	de.Nexthops = make([]iface.ID, int(entry.nNexthops))
	for i := range de.Nexthops {
		de.Nexthops[i] = iface.ID(entry.nexthops[i])
	}

	de.Strategy = strategycode.FromPtr(unsafe.Pointer(*entry.ptrStrategy())).ID()
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

func (entry *Entry) assignReal(u *fibdef.RealUpdate, nDyns int) {
	entry.height = 0

	nameV, _ := u.Name.MarshalBinary()
	entry.nameL = C.uint16_t(copy(cptr.AsByteSlice(&entry.nameV), nameV))
	entry.nComps = C.uint8_t(len(u.Name))

	entry.nNexthops = C.uint8_t(len(u.Nexthops))
	for i, nh := range u.Nexthops {
		entry.nexthops[i] = C.FaceID(nh)
	}

	*entry.ptrStrategy() = (*C.StrategyCode)(strategycode.Get(u.Strategy).Ptr())

	if len(u.Scratch) > 0 {
		for i := 0; i < nDyns; i++ {
			dyn := C.FibEntry_PtrDyn(entry.ptr(), C.int(i))
			copy(cptr.AsByteSlice(&dyn.scratch), u.Scratch)
		}
	}
}

func (entry *Entry) assignVirt(u *fibdef.VirtUpdate, real *Entry) {
	entry.height = C.uint8_t(u.Height)

	nameV, _ := u.Name.MarshalBinary()
	entry.nameL = C.uint16_t(copy(cptr.AsByteSlice(&entry.nameV), nameV))
	entry.nComps = C.uint8_t(len(u.Name))

	*entry.ptrReal() = real.ptr()
}
