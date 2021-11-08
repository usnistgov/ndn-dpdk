// Package fib implements the Forwarding Information Base.
package fib

import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibreplica"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibtree"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"go4.org/must"
)

// Fib represents a Forwarding Information Base (FIB).
type Fib struct {
	tree     *fibtree.Tree
	replicas map[eal.NumaSocket]*fibreplica.Table
}

// Len returns number of entries.
func (fib *Fib) Len() int {
	return fib.tree.CountEntries()
}

// Replica returns replica on specified NUMA socket.
func (fib *Fib) Replica(socket eal.NumaSocket) *fibreplica.Table {
	return fib.replicas[eal.RewriteAnyNumaSocketFirst.Rewrite(socket)]
}

// Close frees the FIB.
func (fib *Fib) Close() (e error) {
	eal.CallMain(fib.doClose)
	return nil
}

func (fib *Fib) doClose() {
	urcu.Barrier() // allow call_rcu to complete; otherwise they could invoke rte_mempool_put on free'd objects
	for _, replica := range fib.replicas {
		if replica != nil {
			must.Close(replica)
		}
	}
}

// List lists entries.
func (fib *Fib) List() (list []Entry) {
	for _, entry := range fib.tree.List() {
		list = append(list, Entry{
			Entry: entry,
			fib:   fib,
		})
	}
	return
}

// Find retrieves an entry by exact match.
func (fib *Fib) Find(name ndn.Name) *Entry {
	entry := fib.tree.Find(name)
	if entry == nil {
		return nil
	}
	return &Entry{
		Entry: *entry,
		fib:   fib,
	}
}

// Insert inserts or replaces a FIB entry.
func (fib *Fib) Insert(entry fibdef.Entry) (e error) {
	if e := entry.Validate(); e != nil {
		return fmt.Errorf("entry.Validate: %w", e)
	}

	eal.CallMain(func() {
		e = fib.doUpdate(fib.tree.Insert(entry))
	})
	return e
}

// Erase deletes a FIB entry.
func (fib *Fib) Erase(name ndn.Name) (e error) {
	eal.CallMain(func() {
		e = fib.doUpdate(fib.tree.Erase(name))
	})
	return e
}

func (fib *Fib) doUpdate(tu fibdef.Update) error {
	updates := make(map[*fibreplica.Table]*fibreplica.UpdateCommand)
	for socket, replica := range fib.replicas {
		u, e := replica.PrepareUpdate(tu)
		if e != nil {
			for replica, u := range updates {
				replica.DiscardUpdate(u)
			}
			tu.Revert()
			return fmt.Errorf("replica[%v].PrepareUpdate: %w", socket, e)
		}
		updates[replica] = u
	}

	for replica, u := range updates {
		replica.ExecuteUpdate(u)
	}
	tu.Commit()
	return nil
}

// New creates a Fib.
func New(cfg fibdef.Config, threads []LookupThread) (*Fib, error) {
	cfg.ApplyDefaults()

	fib := &Fib{
		tree:     fibtree.New(cfg.StartDepth),
		replicas: make(map[eal.NumaSocket]*fibreplica.Table),
	}

	threadByNuma := eal.ClassifyByNumaSocket(threads, eal.RewriteAnyNumaSocketFirst).(map[eal.NumaSocket][]LookupThread)
	for socket, ths := range threadByNuma {
		replica, e := fibreplica.New(cfg, len(ths), socket)
		if e != nil {
			fib.doClose()
			return nil, fmt.Errorf("fibreplica.New(%v): %w", socket, e)
		}
		fib.replicas[socket] = replica
	}

	for socket, ths := range threadByNuma {
		replica := fib.replicas[socket].Ptr()
		for i, th := range ths {
			th.SetFib(replica, i)
		}
	}
	return fib, nil
}

// LookupThread represents an entity that can perform FIB lookups, such as a forwarding thread.
type LookupThread interface {
	eal.WithNumaSocket

	SetFib(replica unsafe.Pointer, i int)
}
