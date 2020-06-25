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
type Cs struct {
	pcct.Pcct
}

// FromPcct converts Pcct to Cs.
func FromPcct(pcct *pcct.Pcct) *Cs {
	return (*Cs)(pcct.GetPtr())
}

func (cs *Cs) getPtr() *C.Cs {
	return (*C.Cs)(cs.Pcct.GetPtr())
}

// Close is forbidden.
func (cs *Cs) Close() error {
	panic("Cs.Close() method is explicitly deleted; use Pcct.Close() to close underlying PCCT")
}

// GetCapacity returns capacity of the specified list, in number of entries.
func (cs *Cs) GetCapacity(list ListID) int {
	return int(C.Cs_GetCapacity(cs.getPtr(), C.CsListId(list)))
}

// CountEntries returns number of entries in the specified list.
func (cs *Cs) CountEntries(list ListID) int {
	return int(C.Cs_CountEntries(cs.getPtr(), C.CsListId(list)))
}

type iPitFindResult interface {
	CopyToCPitFindResult(ptr unsafe.Pointer)
}

// Insert inserts a CS entry by replacing a PIT entry with same key.
func (cs *Cs) Insert(data *ndni.Data, pitFound iPitFindResult) {
	var pitFoundC C.PitFindResult
	pitFound.CopyToCPitFindResult(unsafe.Pointer(&pitFoundC))
	C.Cs_Insert(cs.getPtr(), (*C.Packet)(data.GetPacket().GetPtr()), pitFoundC)
}

// Erase erases a CS entry.
func (cs *Cs) Erase(entry *Entry) {
	C.Cs_Erase(cs.getPtr(), entry.getPtr())
}

// ReadDirectArcP returns direct entries ARC algorithm 'p' variable.
func (cs *Cs) ReadDirectArcP() float64 {
	return float64(C.Cs_GetPriv(cs.getPtr()).directArc.p)
}
