package pit

/*
#include "../pcct/pit.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/container/cs"
	"ndn-dpdk/container/fib"
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
	panic("Cs.Close() method is explicitly deleted; use Pcct.Close() to close underlying PCCT")
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
func (pit Pit) Insert(interest *ndn.Interest, fibEntry *fib.Entry) (pitEntry *Entry, csEntry *cs.Entry) {
	res := C.Pit_Insert(pit.getPtr(), (*C.Packet)(interest.GetPacket().GetPtr()),
		(*C.FibEntry)(unsafe.Pointer(fibEntry)))
	switch C.PitResult_GetKind(res) {
	case C.PIT_INSERT_PIT0, C.PIT_INSERT_PIT1:
		pitEntry = &Entry{C.PitInsertResult_GetPitEntry(res), pit}
	case C.PIT_INSERT_CS:
		csEntry1 := pit.getCs().EntryFromPtr(unsafe.Pointer(C.PitInsertResult_GetCsEntry(res)))
		csEntry = &csEntry1
	}
	return
}

// Erase a PIT entry.
func (pit Pit) Erase(entry Entry) {
	C.Pit_Erase(pit.getPtr(), entry.c)
	entry.c = nil
}

// Result of Pit.FindByData.
type FindResult struct {
	resC C.PitResult
	pit  Pit
}

// Copy to *C.PitResult for use in another package.
func (fr FindResult) CopyToCPitResult(ptr unsafe.Pointer) {
	dst := (*C.PitResult)(ptr)
	dst.ptr = fr.resC.ptr
}

// Determine how many PIT entries are matched.
func (fr FindResult) Len() int {
	switch C.PitResult_GetKind(fr.resC) {
	case C.PIT_FIND_PIT0, C.PIT_FIND_PIT1:
		return 1
	case C.PIT_FIND_PIT01:
		return 2
	}
	return 0
}

// Access matched PIT entries.
func (fr FindResult) GetEntries() (entries []Entry) {
	entries = make([]Entry, 0, 2)
	entry0 := C.PitFindResult_GetPitEntry0(fr.resC)
	if entry0 != nil {
		entries = append(entries, fr.pit.EntryFromPtr(unsafe.Pointer(entry0)))
	}
	entry1 := C.PitFindResult_GetPitEntry1(fr.resC)
	if entry1 != nil {
		entries = append(entries, fr.pit.EntryFromPtr(unsafe.Pointer(entry1)))
	}
	return entries
}

// Find PIT entries matching a Data.
func (pit Pit) FindByData(data *ndn.Data) FindResult {
	resC := C.Pit_FindByData(pit.getPtr(), (*C.Packet)(data.GetPacket().GetPtr()))
	return FindResult{resC, pit}
}

// Find PIT entries matching a Nack.
func (pit Pit) FindByNack(nack *ndn.Nack) *Entry {
	entryC := C.Pit_FindByNack(pit.getPtr(), (*C.Packet)(nack.GetPacket().GetPtr()))
	if entryC == nil {
		return nil
	}
	return &Entry{entryC, pit}
}
