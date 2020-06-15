package fib

/*
#include "../../csrc/fib/fib.h"
*/
import "C"
import (
	"errors"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/fib/fibtree"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

type Config struct {
	Id         string
	MaxEntries int // Entries per partition.
	NBuckets   int // Hashtable buckets.
	StartDepth int // 'M' in 2-stage LPM algorithm.
}

// The FIB.
type Fib struct {
	cfg      Config
	ndt      *ndt.Ndt
	parts    []*partition
	tree     *fibtree.Tree
	commands chan command
}

func New(cfg Config, ndt *ndt.Ndt, numaSockets []eal.NumaSocket) (fib *Fib, e error) {
	if cfg.StartDepth <= ndt.GetPrefixLen() {
		return nil, errors.New("FIB StartDepth must be greater than NDT PrefixLen")
	}

	fib = new(Fib)
	fib.cfg = cfg
	fib.ndt = ndt

	for i, numaSocket := range numaSockets {
		part, e := newPartition(fib, i, numaSocket)
		if e != nil {
			fib.doClose(nil)
			return nil, e
		}
		fib.parts = append(fib.parts, part)
	}

	fib.tree = fibtree.New(cfg.StartDepth, ndt.GetPrefixLen(), ndt.CountElements(),
		func(name *ndn.Name) uint64 { return ndt.GetIndex(ndt.ComputeHash(name)) })

	fib.commands = make(chan command)
	go fib.commandLoop()

	return fib, nil
}

// Get number of entries.
func (fib *Fib) Len() int {
	return fib.tree.CountEntries()
}

// Get *C.Fib pointer for specified partition.
func (fib *Fib) GetPtr(partition int) (ptr unsafe.Pointer) {
	if partition >= 0 && partition < len(fib.parts) {
		ptr = unsafe.Pointer(fib.parts[partition].c)
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
	urcu.Barrier() // allow call_rcu to complete; otherwise they could invoke rte_mempool_put on free'd objects
	for _, part := range fib.parts {
		if part != nil {
			part.Close()
		}
	}
	return nil
}
