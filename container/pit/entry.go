package pit

/*
#include "../../csrc/pcct/pit-entry.h"
#include "../../csrc/pcct/pit.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Entry represents a PIT entry.
type Entry C.PitEntry

// EntryFromPtr converts *C.PitEntry to Entry.
func EntryFromPtr(ptr unsafe.Pointer) *Entry {
	return (*Entry)(ptr)
}

// GetPtr returns *C.PitEntry pointer.
func (entry *Entry) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(entry)
}

func (entry *Entry) getPtr() *C.PitEntry {
	return (*C.PitEntry)(entry)
}

func (entry *Entry) getPitPtr() *C.Pit {
	pccEntryC := C.PccEntry_FromPitEntry(entry.getPtr())
	mempoolC := C.rte_mempool_from_obj(unsafe.Pointer(pccEntryC))
	return (*C.Pit)(unsafe.Pointer(mempoolC))
}

// GetToken returns the PIT token assigned to this entry.
func (entry *Entry) GetToken() uint64 {
	return uint64(C.PitEntry_GetToken(entry.getPtr()))
}

// GetFibSeqNum returns the FIB insertion sequence number recorded in this entry.
func (entry *Entry) GetFibSeqNum() uint32 {
	return uint32(entry.getPtr().fibSeqNum)
}

// ListDns returns downstream records.
func (entry *Entry) ListDns() (list []Dn) {
	c := entry.getPtr()
	list = make([]Dn, 0, C.PIT_ENTRY_MAX_DNS)
	for i := 0; i < int(C.PIT_ENTRY_MAX_DNS); i++ {
		dnC := &c.dns[i]
		if dnC.face == C.FACEID_INVALID {
			return list
		}
		list = append(list, Dn{dnC, entry})
	}
	for extC := c.ext; extC != nil; extC = extC.next {
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

// InsertDn inserts new downstream record, or update existing downstream record.
func (entry *Entry) InsertDn(interest *ndni.Interest) *Dn {
	npktC := (*C.Packet)(interest.GetPacket().GetPtr())
	dnC := C.PitEntry_InsertDn(entry.getPtr(), entry.getPitPtr(), npktC)
	return &Dn{dnC, entry}
}

// ListUps returns upstream records.
func (entry *Entry) ListUps() (list []Up) {
	c := entry.getPtr()
	list = make([]Up, 0, C.PIT_ENTRY_MAX_UPS)
	for i := 0; i < int(C.PIT_ENTRY_MAX_UPS); i++ {
		upC := &c.ups[i]
		if upC.face == C.FACEID_INVALID {
			return list
		}
		list = append(list, Up{upC, entry})
	}
	for extC := c.ext; extC != nil; extC = extC.next {
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
