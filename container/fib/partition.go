package fib

/*
#include "fib.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
)

// Collect partition numbers into a list for logging purpose.
func listPartitionNumbers(parts []*partition) (list []int) {
	for _, part := range parts {
		list = append(list, part.index)
	}
	return list
}

// FIB partition in one forwarding thread.
// Init and Close methods are non-thread-safe.
// All other methods require the caller to have URCU read-side lock.
type partition struct {
	fib          *Fib
	index        int
	c            *C.Fib
	dynMp        dpdk.Mempool
	nEntries     int
	insertSeqNum uint32
}

// Allocate C structs in FIB partition.
func newPartition(fib *Fib, index int, numaSocket dpdk.NumaSocket) (part *partition, e error) {
	part = new(partition)
	part.fib = fib
	part.index = index

	part.dynMp, e = dpdk.NewMempool(fmt.Sprintf("%s_%dd", fib.cfg.Id, index), fib.cfg.MaxEntries, 0,
		int(C.sizeof_FibEntryDyn), numaSocket)
	if e != nil {
		return nil, e
	}

	idC := C.CString(fmt.Sprintf("%s_%de", fib.cfg.Id, index))
	defer C.free(unsafe.Pointer(idC))
	part.c = C.Fib_New(idC, C.uint32_t(fib.cfg.MaxEntries), C.uint32_t(fib.cfg.NBuckets),
		C.unsigned(numaSocket), C.uint8_t(fib.cfg.StartDepth))
	if part.c == nil {
		part.dynMp.Close()
		return nil, dpdk.GetErrno()
	}

	return part, nil
}

// Release C structs in FIB partition.
func (part *partition) Close() error {
	C.Fib_Close(part.c)
	part.c = nil
	part.dynMp.Close()
	return nil
}

// Allocate an unused entry.
func (part *partition) Alloc() (entry *C.FibEntry) {
	if !bool(C.Fib_AllocBulk(part.c, &entry, 1)) {
		return nil
	}
	*entry = C.FibEntry{}
	return entry
}

// Find an entry.
func (part *partition) Find(name *ndn.Name) *C.FibEntry {
	nameV := name.GetValue()
	hash := name.ComputeHash()
	return C.Fib_Find_(part.c, C.uint16_t(len(nameV)), (*C.uint8_t)(nameV.GetPtr()),
		C.uint64_t(hash))
}

// Insert an entry.
func (part *partition) Insert(entryC *C.FibEntry) (isNew bool) {
	part.insertSeqNum++
	entryC.seqNum = C.uint32_t(part.insertSeqNum)

	isNew = bool(C.Fib_Insert(part.c, entryC))
	if isNew {
		part.nEntries++
	}
	return isNew
}

// Erase an entry.
func (part *partition) Erase(entryC *C.FibEntry) {
	entryC.shouldFreeDyn = true
	C.Fib_Erase(part.c, entryC)
	part.nEntries--
}
