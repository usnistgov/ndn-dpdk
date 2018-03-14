package pit

/*
#include "../pcct/pit.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/container/cs"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/ndn"
)

// The Pending Interest Table (PIT).
type Pit struct {
	*pcct.Pcct
}

func (pit Pit) getPtr() *C.Pit {
	return (*C.Pit)(pit.GetPtr())
}

func (pit Pit) getPriv() *C.PitPriv {
	return C.Pit_GetPriv(pit.getPtr())
}

func (pit Pit) getCs() cs.Cs {
	return cs.Cs{pit.Pcct}
}

func (pit Pit) Close() error {
	return nil
}

// Count number of PIT entries.
func (pit Pit) Len() int {
	return int(C.Pit_CountEntries(pit.getPtr()))
}

// Trigger the internal timeout scheduler.
func (pit Pit) TriggerTimeoutSched() {
	C.MinSched_Trigger(pit.getPriv().timeoutSched)
}

// Insert or find a PIT entry for the given Interest.
func (pit Pit) Insert(interest *ndn.Interest) (pitEntry *Entry, csEntry *cs.Entry) {
	insertRes := C.Pit_Insert(pit.getPtr(), (*C.PInterest)(interest.GetPInterestPtr()))
	switch C.PitInsertResult_GetKind(insertRes) {
	case C.PIT_INSERT_PIT0, C.PIT_INSERT_PIT1:
		pitEntry = &Entry{C.PitInsertResult_GetPitEntry(insertRes), pit}
	case C.PIT_INSERT_CS:
		csEntry1 := pit.getCs().EntryFromPtr(unsafe.Pointer(C.PitInsertResult_GetCsEntry(insertRes)))
		csEntry = &csEntry1
	}
	return
}

// Erase a PIT entry.
func (pit Pit) Erase(entry Entry) {
	C.Pit_Erase(pit.getPtr(), entry.c)
	entry.c = nil
}

// Find a PIT entry for the given token.
func (pit Pit) Find(token uint64) (matches []*Entry) {
	var found C.PitFindResult
	C.Pit_Find(pit.getPtr(), C.uint64_t(token), &found)
	matches = make([]*Entry, int(found.nMatches))
	for i := range matches {
		matches[i] = &Entry{found.matches[i], pit}
	}
	return matches
}
