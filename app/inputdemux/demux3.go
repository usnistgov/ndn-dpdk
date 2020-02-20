package inputdemux

/*
#include "demux.h"
*/
import "C"

import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

type Demux3 struct {
	c *C.InputDemux3
}

func NewDemux3(socket dpdk.NumaSocket) Demux3 {
	return Demux3FromPtr(dpdk.ZmallocAligned("InputDemux3", C.sizeof_InputDemux3, 1, socket))
}

func Demux3FromPtr(ptr unsafe.Pointer) (demux3 Demux3) {
	demux3.c = (*C.InputDemux3)(ptr)
	return demux3
}

func (demux3 Demux3) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(demux3.c)
}

func (demux3 Demux3) Close() error {
	dpdk.Free(demux3.GetPtr())
	return nil
}

func (demux3 Demux3) GetInterestDemux() (demux Demux) {
	demux.c = &demux3.c.interest
	return demux
}

func (demux3 Demux3) GetDataDemux() (demux Demux) {
	demux.c = &demux3.c.data
	return demux
}

func (demux3 Demux3) GetNackDemux() (demux Demux) {
	demux.c = &demux3.c.nack
	return demux
}

var Demux3_FaceRx = unsafe.Pointer(C.InputDemux3_FaceRx)
