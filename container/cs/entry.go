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

func (entry *Entry) GetData() *ndn.Data {
	return ndn.PacketFromPtr(unsafe.Pointer(entry.c.data)).AsData()
}

func (entry *Entry) GetFreshUntil() dpdk.TscTime {
	return dpdk.TscTime(entry.c.freshUntil)
}

// Determine whether entry is fresh at a timestamp.
func (entry *Entry) IsFresh(now dpdk.TscTime) bool {
	return entry.GetFreshUntil() > now
}
