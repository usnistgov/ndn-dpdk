package pit

/*
#include "../../csrc/pcct/pit-entry.h"
#include "../../csrc/pcct/pit.h"
enum { c_offsetof_PitEntry_fibSeqNum = offsetof(PitEntry, fibSeqNum) };
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

// Ptr returns *C.PitEntry pointer.
func (entry *Entry) Ptr() unsafe.Pointer {
	return unsafe.Pointer(entry)
}

func (entry *Entry) ptr() *C.PitEntry {
	return (*C.PitEntry)(entry)
}

func (entry *Entry) pitPtr() *C.Pit {
	mempoolC := C.rte_mempool_from_obj(unsafe.Pointer(entry.pccEntry))
	pcctC := (*C.Pcct)(C.rte_mempool_get_priv(mempoolC))
	return &pcctC.pit
}

// PitToken returns the PIT token assigned to this entry.
func (entry *Entry) PitToken() uint64 {
	return uint64(C.PitEntry_GetToken(entry.ptr()))
}

// FibSeqNum returns the FIB insertion sequence number recorded in this entry.
func (entry *Entry) FibSeqNum() uint32 {
	return *(*uint32)(unsafe.Add(entry.Ptr(), C.c_offsetof_PitEntry_fibSeqNum))
}

// DnRecords returns downstream records.
func (entry *Entry) DnRecords() (list []DnRecord) {
	c := entry.ptr()
	list = make([]DnRecord, 0, C.PitMaxDns)
	for i := 0; i < C.PitMaxDns; i++ {
		dnC := &c.dns[i]
		if dnC.face == 0 {
			return list
		}
		list = append(list, DnRecord{dnC, entry})
	}
	for extC := c.ext; extC != nil; extC = extC.next {
		for i := 0; i < C.PitMaxExtDns; i++ {
			dnC := &extC.dns[i]
			if dnC.face == 0 {
				return list
			}
			list = append(list, DnRecord{dnC, entry})
		}
	}
	return list
}

// InsertDnRecord inserts new downstream record, or update existing downstream record.
func (entry *Entry) InsertDnRecord(interest *ndni.Packet) *DnRecord {
	dnC := C.PitEntry_InsertDn(entry.ptr(), entry.pitPtr(), (*C.Packet)(interest.Ptr()))
	return &DnRecord{dnC, entry}
}

// UpRecords returns upstream records.
func (entry *Entry) UpRecords() (list []UpRecord) {
	c := entry.ptr()
	list = make([]UpRecord, 0, C.PitMaxUps)
	for i := 0; i < C.PitMaxUps; i++ {
		upC := &c.ups[i]
		if upC.face == 0 {
			return list
		}
		list = append(list, UpRecord{upC, entry})
	}
	for extC := c.ext; extC != nil; extC = extC.next {
		for i := 0; i < C.PitMaxExtUps; i++ {
			upC := &extC.ups[i]
			if upC.face == 0 {
				return list
			}
			list = append(list, UpRecord{upC, entry})
		}
	}
	return list
}
