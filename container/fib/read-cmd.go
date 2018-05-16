package fib

/*
#include "fib.h"
*/
import "C"
import (
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/ndn"
)

// List all FIB entry names.
func (fib *Fib) ListNames() (names []*ndn.Name) {
	fib.postCommand(func(rs *urcu.ReadSide) error {
		names = make([]*ndn.Name, 0)
		fib.treeRoot.Walk(nodeName{}, func(nn nodeName, node *node) {
			if node.IsEntry {
				names = append(names, nn.GetName())
			}
		})
		return nil
	})
	return names
}

func findC(fibC *C.Fib, nameV ndn.TlvBytes) (entryC *C.FibEntry) {
	return C.__Fib_Find(fibC, C.uint16_t(len(nameV)), (*C.uint8_t)(nameV.GetPtr()))
}

// Perform an exact match lookup.
func (fib *Fib) Find(name *ndn.Name) (entry *Entry) {
	fib.postCommand(func(rs *urcu.ReadSide) error {
		_, partition := fib.ndt.Lookup(name)
		entry = fib.FindInPartition(name, int(partition), rs)
		return nil
	})
	return entry
}

// Perform an exact match lookup in specified partition.
// This method runs in the given URCU read-side thread, not necessarily the command loop.
func (fib *Fib) FindInPartition(name *ndn.Name, partition int, rs *urcu.ReadSide) (entry *Entry) {
	rs.Lock()
	defer rs.Unlock()
	entryC := findC(fib.c[partition], name.GetValue())
	if entryC != nil {
		entry = &Entry{*entryC}
	}
	return entry
}

// Perform a longest prefix match lookup.
func (fib *Fib) Lpm(name *ndn.Name) (entry *Entry) {
	fib.postCommand(func(rs *urcu.ReadSide) error {
		rs.Lock()
		defer rs.Unlock()
		_, partition := fib.ndt.Lookup(name)
		entryC := C.__Fib_Lpm(fib.c[partition], (*C.PName)(name.GetPNamePtr()),
			(*C.uint8_t)(name.GetValue().GetPtr()))
		if entryC != nil {
			entry = &Entry{*entryC}
		}
		return nil
	})
	return entry
}
