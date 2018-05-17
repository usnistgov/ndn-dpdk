package fib

/*
#include "fib.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	"ndn-dpdk/container/ndt"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
)

type Config struct {
	Id         string
	MaxEntries int // Entries per partition.
	NBuckets   int // Hashtable buckets.
	StartDepth int // 'M' in 2-stage LPM algorithm.
}

// The FIB.
type Fib struct {
	c          []*C.Fib
	dynMps     []dpdk.Mempool
	commands   chan command
	startDepth int
	ndt        *ndt.Ndt
	treeRoot   *node
	sti        subtreeIndex

	nNodes              int // Nodes in tree.
	nShortEntries       int // Entries with name shorter than NDT PrefixLen.
	nLongEntries        int // Entries with name not shorter than NDT PrefixLen.
	nEntriesC           int // Entries in C.Fib.
	nRelocatingEntriesC int // Duplicate entries due to relocating.
}

func New(cfg Config, ndt *ndt.Ndt, numaSockets []dpdk.NumaSocket) (fib *Fib, e error) {
	if cfg.StartDepth <= ndt.GetPrefixLen() {
		return nil, errors.New("FIB StartDepth must be greater than NDT PrefixLen")
	}

	fib = new(Fib)
	for i, numaSocket := range numaSockets {
		idC := C.CString(fmt.Sprintf("%s_%d", cfg.Id, i))
		defer C.free(unsafe.Pointer(idC))
		fibC := C.Fib_New(idC, C.uint32_t(cfg.MaxEntries), C.uint32_t(cfg.NBuckets),
			C.unsigned(numaSocket), C.uint8_t(cfg.StartDepth))
		if fibC == nil {
			fib.doClose(nil)
			return nil, dpdk.GetErrno()
		}
		fib.c = append(fib.c, fibC)

		dynMp, e := dpdk.NewMempool(fmt.Sprintf("%s_dyn%d", cfg.Id, i), cfg.MaxEntries, 0,
			int(C.sizeof_FibEntryDyn), numaSocket)
		if e != nil {
			fib.doClose(nil)
			return nil, e
		}
		fib.dynMps = append(fib.dynMps, dynMp)
	}

	fib.startDepth = cfg.StartDepth
	fib.ndt = ndt

	fib.treeRoot = newNode()
	fib.nNodes++

	fib.sti = newSubtreeIndex(ndt)

	fib.commands = make(chan command)
	go fib.commandLoop()

	return fib, nil
}

// Get number of partitions.
func (fib *Fib) CountPartitions() int {
	return len(fib.c)
}

func (fib *Fib) Len() int {
	return fib.CountEntries(false)
}

// Get number of entries.
// If an entry name is shorter than NDT PrefixLen, it is duplicated across all partitions.
// Such entry is counted once if withDup is false, or counted multiple times if withDup is true.
func (fib *Fib) CountEntries(withDup bool) int {
	if withDup {
		return fib.nShortEntries*fib.CountPartitions() + fib.nLongEntries
	}
	return fib.nShortEntries + fib.nLongEntries
}

// Get number of virtual entries.
func (fib *Fib) CountVirtuals() int {
	return fib.nEntriesC - fib.nRelocatingEntriesC - fib.CountEntries(true)
}

// Get *C.Fib pointer for specified partition.
func (fib *Fib) GetPtr(partition int) (ptr unsafe.Pointer) {
	if partition >= 0 && partition < len(fib.c) {
		ptr = unsafe.Pointer(fib.c[partition])
	}
	return ptr
}

type command struct {
	f    func(rs *urcu.ReadSide) error
	done chan<- error
}

// An RCU read-side thread to execute all commands.
func (fib *Fib) commandLoop() {
	rs := urcu.NewReadSide()
	defer rs.Close()
	rs.Offline()
	for cmd, ok := <-fib.commands; ok; cmd, ok = <-fib.commands {
		rs.Online()
		cmd.done <- cmd.f(rs)
		rs.Offline()
	}
}

// Execute a command in the commandLoop thread.
func (fib *Fib) postCommand(f func(rs *urcu.ReadSide) error) error {
	done := make(chan error)
	fib.commands <- command{f: f, done: done}
	return <-done
}

func (fib *Fib) Close() (e error) {
	e = fib.postCommand(fib.doClose)
	close(fib.commands)
	return e
}

func (fib *Fib) doClose(rs *urcu.ReadSide) error {
	for _, fibC := range fib.c {
		C.Fib_Close(fibC)
	}
	for _, dynMp := range fib.dynMps {
		dynMp.Close()
	}
	return nil
}
