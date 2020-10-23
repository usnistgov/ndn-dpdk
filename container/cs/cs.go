// Package cs implements the Content Store.
package cs

/*
#include "../../csrc/pcct/cs.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Cs represents a Content Store (CS).
type Cs C.Cs

// FromPcct converts Pcct to Cs.
func FromPcct(pcct *pcct.Pcct) *Cs {
	pcctC := (*C.Pcct)(pcct.Ptr())
	return (*Cs)(&pcctC.cs)
}

func (cs *Cs) ptr() *C.Cs {
	return (*C.Cs)(cs)
}

// Capacity returns capacity of the specified list, in number of entries.
func (cs *Cs) Capacity(list ListID) int {
	return int(C.Cs_GetCapacity(cs.ptr(), C.CsListID(list)))
}

// CountEntries returns number of entries in the specified list.
func (cs *Cs) CountEntries(list ListID) int {
	return int(C.Cs_CountEntries(cs.ptr(), C.CsListID(list)))
}

type pitFindResult interface {
	CopyToCPitFindResult(ptr unsafe.Pointer)
}

// Insert inserts a CS entry by replacing a PIT entry with same key.
func (cs *Cs) Insert(data *ndni.Packet, pitFound pitFindResult) {
	var pitFoundC C.PitFindResult
	pitFound.CopyToCPitFindResult(unsafe.Pointer(&pitFoundC))
	C.Cs_Insert(cs.ptr(), (*C.Packet)(data.Ptr()), pitFoundC)
}

// Erase erases a CS entry.
func (cs *Cs) Erase(entry *Entry) {
	C.Cs_Erase(cs.ptr(), entry.ptr())
}

// ReadDirectArcP returns direct entries ARC algorithm 'p' variable (for unit testing).
func (cs *Cs) ReadDirectArcP() float64 {
	return float64(cs.ptr().direct.p)
}
