package cs

/*
#include "../../csrc/pcct/cs-entry.h"

static void* CsEntry_Data(CsEntry* entry) { return entry->data; }
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Entry represents a CS entry.
type Entry C.CsEntry

// EntryFromPtr converts *C.CsEntry to Entry.
func EntryFromPtr(ptr unsafe.Pointer) *Entry {
	return (*Entry)(ptr)
}

func (entry *Entry) ptr() *C.CsEntry {
	return (*C.CsEntry)(entry)
}

// Kind returns entry kind.
func (entry *Entry) Kind() EntryKind {
	return EntryKind(entry.kind)
}

// ListIndirects returns a list of indirect entries associated with this direct entry.
// Panics if this is not a direct entry.
func (entry *Entry) ListIndirects() (indirects []*Entry) {
	if entry.Kind() == EntryIndirect {
		panic("Entry.ListIndirects is unavailable on indirect entry")
	}

	c := entry.ptr()
	indirects = make([]*Entry, c.nIndirects)
	for i := range indirects {
		indirects[i] = (*Entry)(c.indirect[i])
	}
	return indirects
}

// Data returns the Data packet on this entry.
func (entry *Entry) Data() *ndni.Packet {
	if entry.Kind() != EntryMemory {
		panic("Entry.Data is only available on in-memory entry")
	}
	return ndni.PacketFromPtr(C.CsEntry_Data((*C.CsEntry)(entry)))
}

// FreshUntil returns a timestamp when this entry would become non-fresh.
func (entry *Entry) FreshUntil() eal.TscTime {
	return eal.TscTime(C.CsEntry_GetDirect(entry.ptr()).freshUntil)
}

// IsFresh determines whether entry is fresh at the given time.
func (entry *Entry) IsFresh(now eal.TscTime) bool {
	return entry.FreshUntil() > now
}
