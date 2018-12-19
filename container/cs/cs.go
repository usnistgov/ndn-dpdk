package cs

/*
#include "../pcct/cs.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/container/pcct"
	"ndn-dpdk/ndn"
)

type ListId int

const (
	CSL_MD = ListId(C.CSL_MD)
	CSL_MI = ListId(C.CSL_MI)
)

// The Content Store (CS).
type Cs struct {
	*pcct.Pcct
}

func (cs Cs) getPtr() *C.Cs {
	return (*C.Cs)(cs.GetPtr())
}

func (cs Cs) Close() error {
	panic("Cs.Close() method is explicitly deleted; use Pcct.Close() to close underlying PCCT")
}

// Get capacity in number of entries.
func (cs Cs) GetCapacity(cslId ListId) int {
	return int(C.Cs_GetCapacity(cs.getPtr(), C.CsListId(cslId)))
}

// Get number of entries.
func (cs Cs) CountEntries(cslId ListId) int {
	return int(C.Cs_CountEntries(cs.getPtr(), C.CsListId(cslId)))
}

type iPitFindResult interface {
	CopyToCPitFindResult(ptr unsafe.Pointer)
}

// Insert a CS entry by replacing a PIT entry with same key.
func (cs Cs) Insert(data *ndn.Data, pitFound iPitFindResult) {
	var pitFoundC C.PitFindResult
	pitFound.CopyToCPitFindResult(unsafe.Pointer(&pitFoundC))
	C.Cs_Insert(cs.getPtr(), (*C.Packet)(data.GetPacket().GetPtr()), pitFoundC)
}

// Erase a CS entry.
func (cs Cs) Erase(entry Entry) {
	C.Cs_Erase(cs.getPtr(), entry.c)
	entry.c = nil
}
