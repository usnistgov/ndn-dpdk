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
func (cs Cs) GetCapacity() int {
	return int(C.Cs_GetCapacity(cs.getPtr()))
}

// Set capacity in number of entries.
func (cs Cs) SetCapacity(capacity int) {
	C.Cs_SetCapacity(cs.getPtr(), C.uint32_t(capacity))
}

// Get number of CS entries.
func (cs Cs) Len() int {
	return int(C.Cs_CountEntries(cs.getPtr()))
}

// Enumerate all CS entries.
func (cs Cs) List() (list []Entry) {
	list = make([]Entry, 0, cs.Len())
	head := &C.Cs_GetPriv(cs.getPtr()).head
	for node := head.next; node != head; node = node.next {
		list = append(list, cs.EntryFromPtr(unsafe.Pointer(node)))
	}
	return list
}

type iPitFindResult interface {
	CopyToCPitResult(ptr unsafe.Pointer)
}

// Insert a CS entry by replacing a PIT entry with same key.
func (cs Cs) Insert(data *ndn.Data, pitFound iPitFindResult) {
	var pitFoundC C.PitResult
	pitFound.CopyToCPitResult(unsafe.Pointer(&pitFoundC))
	C.Cs_Insert(cs.getPtr(), (*C.Packet)(data.GetPacket().GetPtr()), pitFoundC)
}

// Erase a CS entry.
func (cs Cs) Erase(entry Entry) {
	C.Cs_Erase(cs.getPtr(), entry.c)
	entry.c = nil
}
