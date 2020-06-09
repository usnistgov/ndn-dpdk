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
	fib      *Fib
	index    int
	c        *C.Fib
	nEntries int
}

// Allocate C structs in FIB partition.
func newPartition(fib *Fib, index int, numaSocket dpdk.NumaSocket) (part *partition, e error) {
	part = new(partition)
	part.fib = fib
	part.index = index

	idC := C.CString(fmt.Sprintf("%s_%d", fib.cfg.Id, index))
	defer C.free(unsafe.Pointer(idC))
	part.c = C.Fib_New(idC, C.uint32_t(fib.cfg.MaxEntries), C.uint32_t(fib.cfg.NBuckets),
		C.unsigned(numaSocket.ID()), C.uint8_t(fib.cfg.StartDepth))
	if part.c == nil {
		return nil, dpdk.GetErrno()
	}

	return part, nil
}

// Release C structs in FIB partition.
func (part *partition) Close() error {
	C.Fib_Close(part.c)
	part.c = nil
	return nil
}

// Allocate an unused entry.
func (part *partition) Alloc(name *ndn.Name) (entry *C.FibEntry) {
	if !bool(C.Fib_AllocBulk(part.c, &entry, 1)) {
		return nil
	}
	entrySetName(entry, name)
	return entry
}

// Retrieve an entry (either virtual or non-virtual).
func (part *partition) Get(name *ndn.Name) *C.FibEntry {
	nameV := name.GetValue()
	hash := name.ComputeHash()
	return C.Fib_Get_(part.c, C.uint16_t(len(nameV)), (*C.uint8_t)(nameV.GetPtr()),
		C.uint64_t(hash))
}

// Insert an entry.
func (part *partition) Insert(entryC *C.FibEntry, freeVirt, freeReal C.Fib_FreeOld) {
	C.Fib_Insert(part.c, entryC, freeVirt, freeReal)
}

// Erase an entry.
func (part *partition) Erase(entryC *C.FibEntry, freeVirt, freeReal C.Fib_FreeOld) {
	C.Fib_Erase(part.c, entryC, freeVirt, freeReal)
}
