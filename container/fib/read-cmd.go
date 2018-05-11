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
		names = fib.tree.List()
		return nil
	})
	return names
}

func (fib *Fib) findC(nameV ndn.TlvBytes) (entryC *C.FibEntry) {
	return C.__Fib_Find(fib.c, C.uint16_t(len(nameV)), (*C.uint8_t)(nameV.GetPtr()))
}

// Perform an exact match lookup.
// The FIB entry will be copied.
func (fib *Fib) Find(name *ndn.Name) (entry *Entry) {
	fib.postCommand(func(rs *urcu.ReadSide) error {
		rs.Lock()
		defer rs.Unlock()
		entryC := fib.findC(name.GetValue())
		if entryC != nil {
			entry = &Entry{*entryC}
		}
		return nil
	})
	return entry
}

// Perform a longest prefix match lookup.
// The FIB entry will be copied.
func (fib *Fib) Lpm(name *ndn.Name) (entry *Entry) {
	fib.postCommand(func(rs *urcu.ReadSide) error {
		rs.Lock()
		defer rs.Unlock()
		entryC := C.__Fib_Lpm(fib.c, (*C.PName)(name.GetPNamePtr()),
			(*C.uint8_t)(name.GetValue().GetPtr()))
		if entryC != nil {
			entry = &Entry{*entryC}
		}
		return nil
	})
	return entry
}
