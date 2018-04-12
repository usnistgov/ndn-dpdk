package ethface

/*
#include "eth-face.h"

void
c_EthFace_RxLoop(Face* face, uint16_t burstSize, void* cb, void* cbarg)
{
	EthFace_RxLoop(face, burstSize, (Face_RxCb)cb, cbarg);
}
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

func SizeofTxHeader() int {
	return int(C.EthFace_SizeofTxHeader())
}

type EthFace struct {
	iface.BaseFace
	rxLoopStopped chan bool
}

func New(port dpdk.EthDev, mempools iface.Mempools) (*EthFace, error) {
	var face EthFace
	id := iface.FaceId(0x1000 | port)
	face.InitBaseFace(id, int(C.sizeof_EthFacePriv), port.GetNumaSocket())

	if ok := C.EthFace_Init(face.getPtr(), (*C.FaceMempools)(mempools.GetPtr())); !ok {
		return nil, dpdk.GetErrno()
	}

	face.rxLoopStopped = make(chan bool)
	iface.Put(&face)
	return &face, nil
}

func (face *EthFace) getPtr() *C.Face {
	return (*C.Face)(face.GetPtr())
}

func (face *EthFace) getPriv() *C.EthFacePriv {
	return (*C.EthFacePriv)(C.Face_GetPriv(face.getPtr()))
}

func (face *EthFace) GetPort() dpdk.EthDev {
	return dpdk.EthDev(face.GetFaceId() | 0x0FFF)
}

func (face *EthFace) Close() error {
	C.EthFace_Close(face.getPtr())
	return nil
}

func (face *EthFace) RxLoop(burstSize int, cb unsafe.Pointer, cbarg unsafe.Pointer) {
	C.c_EthFace_RxLoop(face.getPtr(), C.uint16_t(burstSize), cb, cbarg)
	face.rxLoopStopped <- true
}

func (face *EthFace) StopRxLoop() error {
	privC := face.getPriv()
	privC.stopRxLoop = true
	<-face.rxLoopStopped
	privC.stopRxLoop = false
	return nil
}

func (face *EthFace) ListFacesInRxLoop() []iface.FaceId {
	return []iface.FaceId{face.GetFaceId()}
}
