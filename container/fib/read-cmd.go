package fib

/*
#include "../../csrc/fib/fib.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/container/fib/fibtree"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// ListNames returns a list of FIB entry names.
func (fib *Fib) ListNames() (names []ndn.Name) {
	eal.CallMain(func() {
		fib.tree.Traverse(func(name ndn.Name, n *fibtree.Node) bool {
			if n.IsEntry {
				names = append(names, name)
			}
			return true
		})
	})
	return names
}

// Find performs an exact match lookup.
func (fib *Fib) Find(name ndn.Name) (entry *Entry) {
	eal.CallMain(func() {
		_, partition := fib.ndt.Lookup(name)
		entry = fib.FindInPartition(name, int(partition), eal.MainReadSide)
	})
	return entry
}

// FindInPartition performs an exact match lookup in specified partition.
// This method runs in the given URCU read-side thread, not necessarily the command loop.
func (fib *Fib) FindInPartition(name ndn.Name, partition int, rs *urcu.ReadSide) (entry *Entry) {
	rs.Lock()
	defer rs.Unlock()
	length, value, hash, _ := convertName(name)
	return entryFromPtr(C.Fib_Find_(fib.parts[partition].c, length, value, hash))
}

// Determine what partitions would a name appear in.
// This method is non-thread-safe.
func (fib *Fib) listPartitionsForName(name ndn.Name) (parts []*partition) {
	if len(name) < fib.ndt.PrefixLen() {
		return fib.parts
	}
	_, partition := fib.ndt.Lookup(name)
	return append(parts, fib.parts[partition])
}

// ReadEntryCounters returns entry counters, aggregated across partitions where the entry appears.
func (fib *Fib) ReadEntryCounters(name ndn.Name) (cnt EntryCounters) {
	eal.CallMain(func() {
		for _, part := range fib.listPartitionsForName(name) {
			if entry := fib.FindInPartition(name, part.index, eal.MainReadSide); entry != nil {
				cnt.Add(entry)
			}
		}
	})
	return cnt
}

// Lpm performs a longest prefix match lookup.
func (fib *Fib) Lpm(name ndn.Name) (entry *Entry) {
	eal.CallMain(func() {
		eal.MainReadSide.Lock()
		defer eal.MainReadSide.Unlock()
		_, partition := fib.ndt.Lookup(name)
		_, value, _, pname := convertName(name)
		entry = entryFromPtr(C.Fib_Lpm_(fib.parts[partition].c, pname, value))
	})
	return entry
}
