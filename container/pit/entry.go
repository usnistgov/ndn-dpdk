package pit

/*
#include "../pcct/pit-entry.h"
#include "../pcct/pit.h"
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

func (entry Entry) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(entry.c)
}

func (entry Entry) GetToken() uint64 {
	return uint64(C.Pit_GetEntryToken(entry.pit.getPtr(), entry.c))
}

func (entry Entry) GetFibSeqNo() uint32 {
	return uint32(entry.c.fibSeqNo)
}

// List DN records.
func (entry Entry) ListDns() (list []Dn) {
	list = make([]Dn, 0, C.PIT_ENTRY_MAX_DNS)
	for i := 0; i < int(C.PIT_ENTRY_MAX_DNS); i++ {
		dnC := &entry.c.dns[i]
		if dnC.face == C.FACEID_INVALID {
			return list
		}
		list = append(list, Dn{dnC, entry})
	}
	for extC := entry.c.ext; extC != nil; extC = extC.next {
		for i := 0; i < int(C.PIT_ENTRY_EXT_MAX_DNS); i++ {
			dnC := &extC.dns[i]
			if dnC.face == C.FACEID_INVALID {
				return list
			}
			list = append(list, Dn{dnC, entry})
		}
	}
	return list
}

// Insert new DN record, or update existing DN record.
func (entry Entry) InsertDn(interest *ndn.Interest) *Dn {
	npktC := (*C.Packet)(interest.GetPacket().GetPtr())
	dnC := C.PitEntry_InsertDn(entry.c, entry.pit.getPtr(), npktC)
	return &Dn{dnC, entry}
}

// List UP records.
func (entry Entry) ListUps() (list []Up) {
	list = make([]Up, 0, C.PIT_ENTRY_MAX_UPS)
	for i := 0; i < int(C.PIT_ENTRY_MAX_UPS); i++ {
		upC := &entry.c.ups[i]
		if upC.face == C.FACEID_INVALID {
			return list
		}
		list = append(list, Up{upC, entry})
	}
	for extC := entry.c.ext; extC != nil; extC = extC.next {
		for i := 0; i < int(C.PIT_ENTRY_EXT_MAX_UPS); i++ {
			upC := &extC.ups[i]
			if upC.face == C.FACEID_INVALID {
				return list
			}
			list = append(list, Up{upC, entry})
		}
	}
	return list
}
