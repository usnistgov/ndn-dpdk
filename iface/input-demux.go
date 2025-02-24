package iface

/*
#include "../csrc/iface/input-demux.h"
*/
import "C"

import (
	"math"
	"slices"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/zyedidia/generic"
)

// InputDemux is a demultiplexer for incoming packets of one L3 type.
//
// The zero value drops all packets.
type InputDemux C.InputDemux

// InputDemuxFromPtr converts *C.InputDemux pointer to InputDemux.
func InputDemuxFromPtr(ptr unsafe.Pointer) *InputDemux {
	return (*InputDemux)(ptr)
}

func (demux *InputDemux) ptr() *C.InputDemux {
	return (*C.InputDemux)(demux)
}

// InitDrop configures to drop all packets.
func (demux *InputDemux) InitDrop() {
	demux.dispatch = C.InputDemuxActDrop
}

// InitFirst configures to pass all packets to the first and only destination.
func (demux *InputDemux) InitFirst() {
	demux.InitRoundrobin(1)
}

// InitRoundrobin configures to pass all packets to each destination in a round-robin fashion.
func (demux *InputDemux) InitRoundrobin(nDest int) {
	C.InputDemux_SetDispatchDiv(demux.ptr(), C.uint32_t(nDest), false)
}

// InitGenericHash configures to dispatch according to hash of GenericNameComponents.
func (demux *InputDemux) InitGenericHash(nDest int) {
	C.InputDemux_SetDispatchDiv(demux.ptr(), C.uint32_t(nDest), true)
}

// InitNdt configures to dispatch via NDT lookup.
//
// Caller must Init() the returned NDT querier to link with a valid NDT table and arrange to
// Clear() the NDT querier before freeing the InputDemux or changing dispatch function.
func (demux *InputDemux) InitNdt() *ndt.Querier {
	ndq := C.InputDemux_SetDispatchByNdt(demux.ptr())
	return (*ndt.Querier)(unsafe.Pointer(ndq))
}

// InitToken configures to dispatch according to specified octet in the PIT token.
func (demux *InputDemux) InitToken(offset int) {
	C.InputDemux_SetDispatchByToken(demux.ptr(), C.uint8_t(offset))
}

// SetDest assigns i-th destination.
func (demux *InputDemux) SetDest(i int, q *PktQueue) {
	demux.dest[i].queue = q.ptr()
}

// Dispatch submits a burst of L3 packets for dispatching.
// Returns booleans to indicate rejected packets (they must be freed by caller).
//
//	chunkSize: between 1 and 64, defaults to 64.
func (demux *InputDemux) Dispatch(chunkSize int, vec pktmbuf.Vector) (rejects []bool) {
	if chunkSize == 0 {
		chunkSize = math.MaxInt
	}
	chunkSize = generic.Clamp(chunkSize, 1, 64)

	rejects = make([]bool, 0, len(vec))
	for pkts := range slices.Chunk(vec, chunkSize) {
		mask := C.InputDemux_Dispatch(demux.ptr(), cptr.FirstPtr[*C.Packet](pkts), C.uint16_t(len(pkts)))
		rejects = append(rejects, cptr.ExpandBits(len(pkts), mask)...)
	}
	return
}

// InputDemuxCounters contains InputDemux counters.
type InputDemuxCounters struct {
	NDrops uint64
}

// Counters reads counters.
func (demux *InputDemux) Counters() (cnt InputDemuxCounters) {
	cnt.NDrops = uint64(demux.nDrops)
	return cnt
}

// InputDemuxDestCounters contains counters of an InputDemux destination.
type InputDemuxDestCounters struct {
	NQueued  uint64
	NDropped uint64
}

// DestCounters returns counters of i-th destination.
func (demux *InputDemux) DestCounters(i int) (cnt InputDemuxDestCounters) {
	cnt.NQueued = uint64(demux.dest[i].nQueued)
	cnt.NDropped = uint64(demux.dest[i].nDropped)
	return cnt
}

// WithInputDemuxes is an object that contains per L3 type InputDemux.
type WithInputDemuxes interface {
	// DemuxOf returns InputDemux of specified PktType.
	// t must be a valid L3 type (not ndni.PktFragment).
	DemuxOf(t ndni.PktType) *InputDemux
}
