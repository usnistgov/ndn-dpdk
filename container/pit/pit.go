// Package pit implements the Pending Interest Table.
package pit

/*
#include "../../csrc/pcct/pit.h"

static_assert(offsetof(PitInsertResult, pitEntry) == offsetof(PitInsertResult, csEntry), "");
enum { c_offsetof_PitInsertResult_Entry = offsetof(PitInsertResult, pitEntry) };
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/cs"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibreplica"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/zap"
)

var logger = logging.New("pit")

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
	return int(pit.nEntries)
}

// TriggerTimeoutSched triggers the internal timeout scheduler.
func (pit *Pit) TriggerTimeoutSched() {
	C.MinSched_Trigger(pit.timeoutSched)
}

// Insert attempts to insert a PIT entry for the given Interest.
// It returns either a new or existing PIT entry, or a CS entry that satisfies the Interest.
func (pit *Pit) Insert(interest *ndni.Packet, fibEntry *fibreplica.Entry) (pitEntry *Entry, csEntry *cs.Entry) {
	ir := C.Pit_Insert(pit.ptr(), (*C.Packet)(interest.Ptr()), (*C.FibEntry)(fibEntry.Ptr()))
	entryPtr := *(*unsafe.Pointer)(unsafe.Add(unsafe.Pointer(&ir), C.c_offsetof_PitInsertResult_Entry))
	switch ir.kind {
	case C.PIT_INSERT_PIT:
		pitEntry = (*Entry)(entryPtr)
	case C.PIT_INSERT_CS:
		csEntry = cs.EntryFromPtr(entryPtr)
	}
	return
}

// Erase erases a PIT entry.
func (pit *Pit) Erase(entry *Entry) {
	C.Pit_Erase(pit.ptr(), entry.ptr())
}

// FindByData searches for PIT entries matching a Data.
func (pit *Pit) FindByData(data *ndni.Packet, token uint64) FindResult {
	return FindResult(C.Pit_FindByData(pit.ptr(), (*C.Packet)(data.Ptr()), C.uint64_t(token)))
}

// FindByNack searches for PIT entries matching a Nack.
func (pit *Pit) FindByNack(nack *ndni.Packet, token uint64) *Entry {
	return (*Entry)(C.Pit_FindByNack(pit.ptr(), (*C.Packet)(nack.Ptr()), C.uint64_t(token)))
}

func init() {
	pcct.InitPit = func(_ pcct.Config, pcct *pcct.Pcct) {
		pit := &((*C.Pcct)(pcct.Ptr())).pit
		logger.Info("init",
			zap.Uintptr("pcct", uintptr(unsafe.Pointer(pcct))),
			zap.Uintptr("pit", uintptr(unsafe.Pointer(pit))),
		)
		C.Pit_Init(pit)
	}
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
