// Package cs implements the Content Store.
package cs

/*
#include "../../csrc/pcct/cs.h"
#include "../../csrc/pcct/cs-disk.h"
*/
import "C"
import (
	"errors"
	"unsafe"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/container/disk"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/zap"
)

var logger = logging.New("cs")

// Cs represents a Content Store (CS).
type Cs C.Cs

// FromPcct converts Pcct to Cs.
func FromPcct(pcct *pcct.Pcct) *Cs {
	pcctC := (*C.Pcct)(pcct.Ptr())
	return (*Cs)(&pcctC.cs)
}

func (cs *Cs) ptr() *C.Cs {
	return (*C.Cs)(cs)
}

// Capacity returns capacity of the specified list, in number of entries.
func (cs *Cs) Capacity(list ListID) int {
	return int(C.Cs_GetCapacity(cs.ptr(), C.CsListID(list)))
}

// CountEntries returns number of entries in the specified list.
func (cs *Cs) CountEntries(list ListID) int {
	return int(C.Cs_CountEntries(cs.ptr(), C.CsListID(list)))
}

type pitFindResult interface {
	CopyToCPitFindResult(ptr unsafe.Pointer)
}

// Insert inserts a CS entry by replacing a PIT entry with same key.
func (cs *Cs) Insert(data *ndni.Packet, pitFound pitFindResult) {
	var pitFoundC C.PitFindResult
	pitFound.CopyToCPitFindResult(unsafe.Pointer(&pitFoundC))
	C.Cs_Insert(cs.ptr(), (*C.Packet)(data.Ptr()), pitFoundC)
}

// Erase erases a CS entry.
func (cs *Cs) Erase(entry *Entry) {
	C.Cs_Erase(cs.ptr(), entry.ptr())
}

// ReadDirectArcP returns direct entries ARC algorithm 'p' variable (for unit testing).
func (cs *Cs) ReadDirectArcP() float64 {
	return float64(cs.ptr().direct.p)
}

// SetDisk enables on-disk caching.
func (cs *Cs) SetDisk(store *disk.Store, alloc *disk.Alloc) error {
	sMin, sMax := store.SlotRange()
	aMin, aMax := alloc.SlotRange()
	if sMin > aMin || sMax < aMax {
		return errors.New("DiskAlloc slot range out of bound")
	}

	capAlloc, capB2 := alloc.Capacity(), cs.Capacity(ListDirectB2)
	if capAlloc < capB2 {
		logger.Warn("disk allocator capacity is smaller than CS index capacity reserved for on-disk entries",
			zap.Int("cap-alloc", capAlloc),
			zap.Int("cap-B2", capB2),
		)
	}

	cs.diskStore = (*C.DiskStore)(store.Ptr())
	cs.diskAlloc = (*C.DiskAlloc)(alloc.Ptr())
	cs.direct.moveCb = C.CsArc_MoveCb(C.CsDisk_ArcMove)
	cs.direct.moveCbArg = unsafe.Pointer(cs)

	logger.Info("disk caching enabled",
		zap.Uintptr("cs", uintptr(unsafe.Pointer(cs))),
		zap.Int("cap-B2", capB2),
		zap.Uintptr("store", uintptr(store.Ptr())),
		zap.Uint64s("store-slots", []uint64{sMin, sMax}),
		zap.Uintptr("alloc", uintptr(alloc.Ptr())),
		zap.Uint64s("alloc-slots", []uint64{aMin, aMax}),
		zap.Int("cap-alloc", capAlloc),
	)
	return nil
}

func init() {
	pcct.InitCs = func(cfg pcct.Config, pcct *pcct.Pcct) {
		adjustCapacity := func(v, min, dflt int) int {
			if v <= 0 {
				v = dflt
			}
			return math.MaxInt(v, min)
		}

		capMemory := adjustCapacity(cfg.CsMemoryCapacity, EvictBulk, cfg.PcctCapacity/4)
		capDisk := adjustCapacity(cfg.CsDiskCapacity, 0, 0)
		capIndirect := adjustCapacity(cfg.CsIndirectCapacity, EvictBulk, cfg.PcctCapacity/4)

		cs := &((*C.Pcct)(pcct.Ptr())).cs
		logger.Info("init",
			zap.Uintptr("pcct", uintptr(unsafe.Pointer(pcct))),
			zap.Uintptr("cs", uintptr(unsafe.Pointer(cs))),
			zap.Int("cap-memory", capMemory),
			zap.Int("cap-disk", capDisk),
			zap.Int("cap-indirect", capIndirect),
		)

		C.CsArc_Init(&cs.direct, C.uint32_t(capMemory), C.uint32_t(capDisk))
		C.CsList_Init(&cs.indirect)
		cs.indirect.capacity = C.uint32_t(capIndirect)
	}
}
