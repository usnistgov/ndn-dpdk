package inputdemux

/*
#include "../../csrc/inputdemux/demux.h"
*/
import "C"

import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

type Demux3 C.InputDemux3

func NewDemux3(socket eal.NumaSocket) *Demux3 {
	return Demux3FromPtr(eal.ZmallocAligned("InputDemux3", C.sizeof_InputDemux3, 1, socket))
}

func Demux3FromPtr(ptr unsafe.Pointer) *Demux3 {
	return (*Demux3)(ptr)
}

func (demux3 *Demux3) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(demux3.getPtr())
}

func (demux3 *Demux3) getPtr() *C.InputDemux3 {
	return (*C.InputDemux3)(demux3)
}

func (demux3 *Demux3) Close() error {
	eal.Free(demux3.GetPtr())
	return nil
}

func (demux3 *Demux3) GetInterestDemux() *Demux {
	return (*Demux)(&demux3.getPtr().interest)
}

func (demux3 *Demux3) GetDataDemux() *Demux {
	return (*Demux)(&demux3.getPtr().data)
}

func (demux3 *Demux3) GetNackDemux() *Demux {
	return (*Demux)(&demux3.getPtr().nack)
}

var Demux3_FaceRx = unsafe.Pointer(C.InputDemux3_FaceRx)
