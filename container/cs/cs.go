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
	return nil
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

type iPitEntry interface {
	// Return the *C.PitEntry pointer.
	GetPitEntryPtr() unsafe.Pointer
}

// Insert a CS entry by replacing a PIT entry with same key.
func (cs Cs) ReplacePitEntry(pitEntry iPitEntry, data *ndn.Data) {
	C.Cs_ReplacePitEntry(cs.getPtr(), (*C.PitEntry)(pitEntry.GetPitEntryPtr()),
		(*C.Packet)(data.GetPacket().GetPtr()))
}

// Erase a CS entry.
func (cs Cs) Erase(entry Entry) {
	C.Cs_Erase(cs.getPtr(), entry.c)
	entry.c = nil
}
