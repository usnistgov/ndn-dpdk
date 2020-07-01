package cs

/*
#include "../../csrc/pcct/cs-entry.h"
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

// IsDirect determines whether this is a direct entry.
func (entry *Entry) IsDirect() bool {
	return bool(C.CsEntry_IsDirect(entry.ptr()))
}

// GetDirect returns the direct entry from a possibly indirect entry.
func (entry *Entry) GetDirect() *Entry {
	return (*Entry)(C.CsEntry_GetDirect(entry.ptr()))
}

// ListIndirects returns a list of indirect entries associated with this direct entry.
// Panics if this is not a direct entry.
func (entry *Entry) ListIndirects() (indirects []*Entry) {
	if !entry.IsDirect() {
		panic("Entry.ListIndirects is unavailable on indirect entry")
	}

	c := entry.ptr()
	indirects = make([]*Entry, int(c.nIndirects))
	for i := range indirects {
		indirects[i] = (*Entry)(c.indirect[i])
	}
	return indirects
}

// GetData returns the Data packet on this entry.
func (entry *Entry) GetData() *ndni.Data {
	return ndni.PacketFromPtr(unsafe.Pointer(C.CsEntry_GetData(entry.ptr()))).AsData()
}

// GetFreshUntil returns a timestamp when this entry would become non-fresh.
func (entry *Entry) GetFreshUntil() eal.TscTime {
	return eal.TscTime(C.CsEntry_GetDirect(entry.ptr()).freshUntil)
}

// IsFresh determines whether entry is fresh at the given time.
func (entry *Entry) IsFresh(now eal.TscTime) bool {
	return entry.GetFreshUntil() > now
}
