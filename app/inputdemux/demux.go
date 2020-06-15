package inputdemux

/*
#include "../../csrc/inputdemux/demux.h"
*/
import "C"

import (
	"unsafe"

	"ndn-dpdk/container/ndt"
	"ndn-dpdk/container/pktqueue"
	"ndn-dpdk/dpdk/eal"
)

// Input packet demuxer for a single packet type.
type Demux C.InputDemux

func NewDemux(socket eal.NumaSocket) *Demux {
	return DemuxFromPtr(eal.ZmallocAligned("InputDemux", C.sizeof_InputDemux, 1, socket))
}

func DemuxFromPtr(ptr unsafe.Pointer) *Demux {
	return (*Demux)(ptr)
}

func (demux *Demux) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(demux)
}

func (demux *Demux) getPtr() *C.InputDemux {
	return (*C.InputDemux)(demux)
}

func (demux *Demux) Close() error {
	eal.Free(demux.GetPtr())
	return nil
}

// Configure to drop all packets.
func (demux *Demux) InitDrop() {
	C.InputDemux_SetDispatchFunc_(demux.getPtr(), C.InputDemux_DispatchDrop)
}

// Configure to pass all packets to the first and only destination.
func (demux *Demux) InitFirst() {
	demux.InitRoundrobin(1)
}

// Configure to pass all packets to each destination in a round-robin fashion.
func (demux *Demux) InitRoundrobin(nDest int) {
	C.InputDemux_SetDispatchRoundrobin_(demux.getPtr(), C.uint32_t(nDest))
}

// Configure to dispatch via NDT loopup.
func (demux *Demux) InitNdt(ndt *ndt.Ndt, ndtThreadId int) {
	demuxC := demux.getPtr()
	C.InputDemux_SetDispatchFunc_(demuxC, C.InputDemux_DispatchByNdt)
	demuxC.ndt = (*C.Ndt)(unsafe.Pointer(ndt.GetPtr()))
	demuxC.ndtt = C.Ndt_GetThread(demuxC.ndt, C.uint8_t(ndtThreadId))
}

// Configure to dispatch according to high 8 bits of PIT token.
func (demux *Demux) InitToken() {
	C.InputDemux_SetDispatchFunc_(demux.getPtr(), C.InputDemux_DispatchByToken)
}

func (demux *Demux) SetDest(index int, q *pktqueue.PktQueue) {
	demux.getPtr().dest[index].queue = (*C.PktQueue)(q.GetPtr())
}

type DestCounters struct {
	NQueued  uint64
	NDropped uint64
}

func (demux *Demux) ReadDestCounters(index int) (cnt DestCounters) {
	demuxC := demux.getPtr()
	cnt.NQueued = uint64(demuxC.dest[index].nQueued)
	cnt.NDropped = uint64(demuxC.dest[index].nDropped)
	return cnt
}
