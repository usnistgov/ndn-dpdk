package iface

/*
#include "../csrc/iface/input-demux.h"
*/
import "C"

import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// InputDemux is a demultiplexer for incoming packets of one L3 type.
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
	demux.dispatch = C.InputDemux_DispatchFunc(C.InputDemux_DispatchDrop)
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

// InitNdt configures to dispatch via NDT loopup.
func (demux *InputDemux) InitNdt(ndq *ndt.Querier) {
	C.InputDemux_SetDispatchByNdt(demux.ptr(), (*C.NdtQuerier)(ndq.Ptr()))
}

// InitToken configures to dispatch according to specified octet in the PIT token.
func (demux *InputDemux) InitToken(offset uint8) {
	C.InputDemux_SetDispatchByToken(demux.ptr(), C.uint8_t(offset))
}

// SetDest assigns i-th destination.
func (demux *InputDemux) SetDest(i int, q *PktQueue) {
	demux.dest[i].queue = q.ptr()
}

// Dispatch submits a packet for dispatching.
func (demux *InputDemux) Dispatch(pkt *ndni.Packet) {
	C.InputDemux_Dispatch(demux.ptr(), (*C.Packet)(pkt.Ptr()), (*C.PName)(unsafe.Pointer(pkt.PName())))
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
