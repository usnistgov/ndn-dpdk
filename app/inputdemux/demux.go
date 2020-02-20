package inputdemux

/*
#include "demux.h"
*/
import "C"

import (
	"unsafe"

	"ndn-dpdk/container/ndt"
	"ndn-dpdk/container/pktqueue"
	"ndn-dpdk/dpdk"
)

type Demux struct {
	c *C.InputDemux
}

func NewDemux(socket dpdk.NumaSocket) Demux {
	return DemuxFromPtr(dpdk.ZmallocAligned("InputDemux", C.sizeof_InputDemux, 1, socket))
}

func DemuxFromPtr(ptr unsafe.Pointer) (demux Demux) {
	demux.c = (*C.InputDemux)(ptr)
	return demux
}

func (demux Demux) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(demux.c)
}

func (demux Demux) Close() error {
	dpdk.Free(demux.GetPtr())
	return nil
}

func (demux Demux) InitDrop() {
	C.InputDemux_SetDispatchFunc_(demux.c, C.InputDemux_DispatchDrop)
}

func (demux Demux) InitNdt(ndt *ndt.Ndt, ndtThreadId int) {
	C.InputDemux_SetDispatchFunc_(demux.c, C.InputDemux_DispatchByNdt)
	demux.c.ndt = (*C.Ndt)(unsafe.Pointer(ndt.GetPtr()))
	demux.c.ndtt = C.Ndt_GetThread(demux.c.ndt, C.uint8_t(ndtThreadId))
}

func (demux Demux) InitToken() {
	C.InputDemux_SetDispatchFunc_(demux.c, C.InputDemux_DispatchByToken)
}

func (demux Demux) InitFirst() {
	C.InputDemux_SetDispatchFunc_(demux.c, C.InputDemux_DispatchToFirst)
}

func (demux Demux) SetDest(index int, q pktqueue.PktQueue) {
	demux.c.dest[index].queue = (*C.PktQueue)(q.GetPtr())
}

type DestCounters struct {
	NQueued  uint64
	NDropped uint64
}

func (demux Demux) ReadDestCounters(index int) (cnt DestCounters) {
	cnt.NQueued = uint64(demux.c.dest[index].nQueued)
	cnt.NDropped = uint64(demux.c.dest[index].nDropped)
	return cnt
}
