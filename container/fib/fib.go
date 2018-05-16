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
	MaxEntries int
	NBuckets   int
	StartDepth int
}

// The FIB.
type Fib struct {
	c          []*C.Fib
	commands   chan command
	startDepth int
	ndt        *ndt.Ndt
	treeRoot   *node

	nNodes        int
	nShortEntries int // Entries with name shorter than NDT PrefixLen in tree.
	nLongEntries  int // Entries with name not shorter than NDT PrefixLen in tree.
	nEntriesC     int // Entries in C.Fib.
}

func New(cfg Config, ndt *ndt.Ndt, numaSockets []dpdk.NumaSocket) (fib *Fib, e error) {
	if cfg.StartDepth <= ndt.GetPrefixLen() {
		return nil, errors.New("FIB StartDepth must be greater than NDT PrefixLen")
	}

	fib = new(Fib)
	fib.c = make([]*C.Fib, len(numaSockets))
	for i, numaSocket := range numaSockets {
		idC := C.CString(fmt.Sprintf("%s_%d", cfg.Id, i))
		defer C.free(unsafe.Pointer(idC))
		fib.c[i] = C.Fib_New(idC, C.uint32_t(cfg.MaxEntries), C.uint32_t(cfg.NBuckets),
			C.unsigned(numaSocket), C.uint8_t(cfg.StartDepth))
		if fib.c[i] == nil {
			for i--; i >= 0; i-- {
				C.Fib_Close(fib.c[i])
			}
			return nil, dpdk.GetErrno()
		}
	}

	fib.startDepth = cfg.StartDepth
	fib.ndt = ndt

	fib.treeRoot = newNode()
	fib.nNodes++

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
	return fib.nEntriesC - fib.CountEntries(true)
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
	e = fib.postCommand(func(rs *urcu.ReadSide) error {
		for _, fibC := range fib.c {
			C.Fib_Close(fibC)
		}
		return nil

	})
	close(fib.commands)
	return e
}
