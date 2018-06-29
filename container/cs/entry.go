package cs

/*
#include "../pcct/cs-entry.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
)

// A CS entry.
type Entry struct {
	c  *C.CsEntry
	cs Cs
}

func (cs Cs) EntryFromPtr(ptr unsafe.Pointer) Entry {
	return Entry{(*C.CsEntry)(ptr), cs}
}

func (entry Entry) IsDirect() bool {
	return bool(C.CsEntry_IsDirect(entry.c))
}

func (entry Entry) GetDirect() Entry {
	return Entry{C.CsEntry_GetDirect(entry.c), entry.cs}
}

func (entry Entry) ListIndirects() (indirects []Entry) {
	if !entry.IsDirect() {
		panic("Entry.ListIndirects is unavailable on indirect entry")
	}

	indirects = make([]Entry, int(entry.c.nIndirects))
	for i := range indirects {
		indirects[i] = Entry{entry.c.indirect[i], entry.cs}
	}
	return indirects
}

func (entry Entry) GetData() *ndn.Data {
	return ndn.PacketFromPtr(unsafe.Pointer(C.CsEntry_GetData(entry.c))).AsData()
}

func (entry Entry) GetFreshUntil() dpdk.TscTime {
	return dpdk.TscTime(C.CsEntry_GetDirect(entry.c).freshUntil)
}

// Determine whether entry is fresh at a timestamp.
func (entry Entry) IsFresh(now dpdk.TscTime) bool {
	return entry.GetFreshUntil() > now
}
