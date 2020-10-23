// Package pit implements the Pending Interest Table.
package pit

/*
#include "../../csrc/pcct/pit.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/cs"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibreplica"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Pit represents a Pending Interest Table (PIT).
type Pit C.Pit

// FromPcct converts Pcct to Pit.
func FromPcct(pcct *pcct.Pcct) *Pit {
	pcctC := (*C.Pcct)(pcct.Ptr())
	return (*Pit)(&pcctC.pit)
}

func (pit *Pit) ptr() *C.Pit {
	return (*C.Pit)(pit)
}

// Len returns number of PIT entries.
func (pit *Pit) Len() int {
	return int(pit.ptr().nEntries)
}

// TriggerTimeoutSched triggers the internal timeout scheduler.
func (pit *Pit) TriggerTimeoutSched() {
	C.MinSched_Trigger(pit.ptr().timeoutSched)
}

// Insert attempts to insert a PIT entry for the given Interest.
// It returns either a new or existing PIT entry, or a CS entry that satisfies the Interest.
func (pit *Pit) Insert(interest *ndni.Packet, fibEntry *fibreplica.Entry) (pitEntry *Entry, csEntry *cs.Entry) {
	res := C.Pit_Insert(pit.ptr(), (*C.Packet)(interest.Ptr()), (*C.FibEntry)(fibEntry.Ptr()))
	switch C.PitInsertResult_GetKind(res) {
	case C.PIT_INSERT_PIT0, C.PIT_INSERT_PIT1:
		pitEntry = (*Entry)(C.PitInsertResult_GetPitEntry(res))
	case C.PIT_INSERT_CS:
		csEntry = cs.EntryFromPtr(unsafe.Pointer(C.PitInsertResult_GetCsEntry(res)))
	}
	return
}

// Erase erases a PIT entry.
func (pit *Pit) Erase(entry *Entry) {
	C.Pit_Erase(pit.ptr(), entry.ptr())
}

// FindByData searches for PIT entries matching a Data.
func (pit *Pit) FindByData(data *ndni.Packet) FindResult {
	resC := C.Pit_FindByData(pit.ptr(), (*C.Packet)(data.Ptr()))
	return FindResult(resC)
}

// FindByNack searches for PIT entries matching a Nack.
func (pit *Pit) FindByNack(nack *ndni.Packet) *Entry {
	entryC := C.Pit_FindByNack(pit.ptr(), (*C.Packet)(nack.Ptr()))
	if entryC == nil {
		return nil
	}
	return (*Entry)(entryC)
}

// FindResult represents the result of Pit.FindByData.
type FindResult C.PitFindResult

// CopyToCPitFindResult copies this result to *C.PitFindResult.
func (fr FindResult) CopyToCPitFindResult(ptr unsafe.Pointer) {
	*(*FindResult)(ptr) = fr
}

// ListEntries returns matched PIT entries.
func (fr FindResult) ListEntries() (entries []*Entry) {
	frC := C.PitFindResult(fr)
	entries = make([]*Entry, 0, 2)
	if entry0 := C.PitFindResult_GetPitEntry0(frC); entry0 != nil {
		entries = append(entries, (*Entry)(entry0))
	}
	if entry1 := C.PitFindResult_GetPitEntry1(frC); entry1 != nil {
		entries = append(entries, (*Entry)(entry1))
	}
	return entries
}

// NeedDataDigest returns true if the result indicates that Data digest computation is needed.
func (fr FindResult) NeedDataDigest() bool {
	frC := C.PitFindResult(fr)
	return bool(C.PitFindResult_Is(frC, C.PIT_FIND_NEED_DIGEST))
}
