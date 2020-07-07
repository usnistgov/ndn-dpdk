package fib

/*
#include "../../csrc/fib/fib.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
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
func newPartition(fib *Fib, index int, numaSocket eal.NumaSocket) (part *partition, e error) {
	part = new(partition)
	part.fib = fib
	part.index = index

	idC := C.CString(eal.AllocObjectID("fib.partition"))
	defer C.free(unsafe.Pointer(idC))
	part.c = C.Fib_New(idC, C.uint32_t(fib.cfg.MaxEntries), C.uint32_t(fib.cfg.NBuckets),
		C.uint(numaSocket.ID()), C.uint8_t(fib.cfg.StartDepth))
	if part.c == nil {
		return nil, eal.GetErrno()
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
func (part *partition) Alloc(name ndn.Name) (entry *Entry) {
	if !bool(C.Fib_AllocBulk(part.c, (**C.FibEntry)(unsafe.Pointer(&entry)), 1)) {
		return nil
	}
	entry.SetName(name)
	return entry
}

// Retrieve an entry (either virtual or non-virtual).
func (part *partition) Get(name ndn.Name) *Entry {
	length, value, hash, _ := convertName(name)
	return entryFromPtr(C.Fib_Get_(part.c, length, value, hash))
}

func (part *partition) Insert(entry *Entry, freeVirt, freeReal C.Fib_FreeOld) {
	C.Fib_Insert(part.c, entry.ptr(), freeVirt, freeReal)
}

func (part *partition) Erase(entry *Entry, freeVirt, freeReal C.Fib_FreeOld) {
	C.Fib_Erase(part.c, entry.ptr(), freeVirt, freeReal)
}

func convertName(name ndn.Name) (length C.uint16_t, value *C.uint8_t, hash C.uint64_t, pname *C.PName) {
	cname := ndni.CNameFromName(name)
	length = C.uint16_t(cname.P.NOctets)
	value = (*C.uint8_t)(unsafe.Pointer(cname.V))
	hash = C.uint64_t(cname.ComputeHash())
	pname = (*C.PName)(unsafe.Pointer(&cname.P))
	return
}
