package pit

/*
#include "../../csrc/pcct/pit.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/cs"
	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// The Pending Interest Table (PIT).
type Pit struct {
	pcct.Pcct
}

func FromPcct(pcct *pcct.Pcct) *Pit {
	return (*Pit)(pcct.GetPtr())
}

func (pit *Pit) getPtr() *C.Pit {
	return (*C.Pit)(pit.Pcct.GetPtr())
}

func (pit *Pit) getPriv() *C.PitPriv {
	return C.Pit_GetPriv(pit.getPtr())
}

func (pit *Pit) Close() error {
	panic("Pit.Close() method is explicitly deleted; use Pcct.Close() to close underlying PCCT")
}

// Count number of PIT entries.
func (pit *Pit) Len() int {
	return int(C.Pit_CountEntries(pit.getPtr()))
}

// Trigger the internal timeout scheduler.
func (pit *Pit) TriggerTimeoutSched() {
	C.MinSched_Trigger(pit.getPriv().timeoutSched)
}

// Insert or find a PIT entry for the given Interest.
func (pit *Pit) Insert(interest *ndn.Interest, fibEntry *fib.Entry) (pitEntry *Entry, csEntry *cs.Entry) {
	res := C.Pit_Insert(pit.getPtr(), (*C.Packet)(interest.GetPacket().GetPtr()),
		(*C.FibEntry)(unsafe.Pointer(fibEntry)))
	switch C.PitInsertResult_GetKind(res) {
	case C.PIT_INSERT_PIT0, C.PIT_INSERT_PIT1:
		pitEntry = &Entry{C.PitInsertResult_GetPitEntry(res), pit}
	case C.PIT_INSERT_CS:
		csEntry1 := cs.FromPcct(&pit.Pcct).EntryFromPtr(unsafe.Pointer(C.PitInsertResult_GetCsEntry(res)))
		csEntry = &csEntry1
	}
	return
}

// Erase a PIT entry.
func (pit *Pit) Erase(entry Entry) {
	C.Pit_Erase(pit.getPtr(), entry.c)
	entry.c = nil
}

// Result of Pit.FindByData.
type FindResult struct {
	resC C.PitFindResult
	pit  *Pit
}

// Copy to *C.PitFindResult for use in another package.
func (fr FindResult) CopyToCPitFindResult(ptr unsafe.Pointer) {
	dst := (*C.PitFindResult)(ptr)
	dst.entry = fr.resC.entry
	dst.kind = fr.resC.kind
}

// Access matched PIT entries.
func (fr FindResult) ListEntries() (entries []Entry) {
	entries = make([]Entry, 0, 2)
	if entry0 := C.PitFindResult_GetPitEntry0(fr.resC); entry0 != nil {
		entries = append(entries, fr.pit.EntryFromPtr(unsafe.Pointer(entry0)))
	}
	if entry1 := C.PitFindResult_GetPitEntry1(fr.resC); entry1 != nil {
		entries = append(entries, fr.pit.EntryFromPtr(unsafe.Pointer(entry1)))
	}
	return entries
}

func (fr FindResult) NeedDataDigest() bool {
	return bool(C.PitFindResult_Is(fr.resC, C.PIT_FIND_NEED_DIGEST))
}

// Find PIT entries matching a Data.
func (pit *Pit) FindByData(data *ndn.Data) FindResult {
	resC := C.Pit_FindByData(pit.getPtr(), (*C.Packet)(data.GetPacket().GetPtr()))
	return FindResult{resC, pit}
}

// Find PIT entries matching a Nack.
func (pit *Pit) FindByNack(nack *ndn.Nack) *Entry {
	entryC := C.Pit_FindByNack(pit.getPtr(), (*C.Packet)(nack.GetPacket().GetPtr()))
	if entryC == nil {
		return nil
	}
	return &Entry{entryC, pit}
}
