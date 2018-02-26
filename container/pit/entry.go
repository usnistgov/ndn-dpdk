package pit

/*
#include "../pcct/pit-entry.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/ndn"
)

// A PIT entry.
type Entry struct {
	c   *C.PitEntry
	pit Pit
}

func (pit Pit) EntryFromPtr(ptr unsafe.Pointer) Entry {
	return Entry{(*C.PitEntry)(ptr), pit}
}

// Implements cs.iPitEntry.
func (entry Entry) GetPitEntryPtr() unsafe.Pointer {
	return unsafe.Pointer(entry.c)
}

// Determine whether two Entry instances point to the same underlying entry.
func (entry Entry) SameAs(entry2 Entry) bool {
	return entry.c == entry2.c
}

// List downstream records.
func (entry Entry) ListDns() (list []Dn) {
	list = make([]Dn, 0, C.PIT_ENTRY_MAX_DNS)
	for index := 0; index < int(C.PIT_ENTRY_MAX_DNS); index++ {
		dnC := &entry.c.dns[index]
		if dnC.face == C.FACEID_INVALID {
			break
		}
		list = append(list, Dn{dnC, entry})
	}
	return list
}

// Refresh downstream record for RX Interest.
func (entry Entry) DnRxInterest(interest *ndn.Interest) bool {
	npktC := (*C.Packet)(interest.GetPacket().GetPtr())
	index := C.PitEntry_DnRxInterest(entry.pit.getPtr(), entry.c, npktC)
	return index >= 0
}
