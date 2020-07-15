package fib

/*
#include "../../csrc/fib/fib.h"
*/
import "C"
import (
	"errors"
	"math/rand"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/mempool"
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
	mp       *mempool.Mempool
	c        *C.Fib
	nEntries int
}

// Allocate C structs in FIB partition.
func newPartition(fib *Fib, index int, socket eal.NumaSocket) (part *partition, e error) {
	part = &partition{
		fib:   fib,
		index: index,
	}

	part.mp, e = mempool.New(mempool.Config{
		Capacity:       fib.cfg.MaxEntries,
		ElementSize:    int(C.sizeof_FibEntry),
		PrivSize:       int(C.sizeof_Fib),
		Socket:         socket,
		NoCache:        true,
		SingleProducer: true,
		SingleConsumer: true,
	})
	if e != nil {
		return nil, e
	}
	mpC := (*C.struct_rte_mempool)(part.mp.Ptr())
	part.c = (*C.Fib)(C.rte_mempool_get_priv(mpC))
	part.c.mp = mpC

	part.c.lfht = C.cds_lfht_new(C.ulong(fib.cfg.NBuckets), C.ulong(fib.cfg.NBuckets), C.ulong(fib.cfg.NBuckets), 0, nil)
	if part.c.lfht == nil {
		part.mp.Close()
		return nil, errors.New("cds_lfht_new error")
	}

	part.c.startDepth = C.int(fib.cfg.StartDepth)
	part.c.insertSeqNum = C.uint32_t(rand.Uint32())
	return part, nil
}

// Release C structs in FIB partition.
func (part *partition) Close() error {
	C.Fib_Clear(part.c)
	C.cds_lfht_destroy(part.c.lfht, nil)
	return part.mp.Close()
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
	pname := ndni.NewPName(name)
	defer pname.Free()
	lname := *(*C.LName)(pname.Ptr())
	return entryFromPtr(C.Fib_Get(part.c, lname, C.uint64_t(pname.ComputeHash())))
}

func (part *partition) Insert(entry *Entry, freeVirt, freeReal C.Fib_FreeOld) {
	C.Fib_Insert(part.c, entry.ptr(), freeVirt, freeReal)
}

func (part *partition) Erase(entry *Entry, freeVirt, freeReal C.Fib_FreeOld) {
	C.Fib_Erase(part.c, entry.ptr(), freeVirt, freeReal)
}
