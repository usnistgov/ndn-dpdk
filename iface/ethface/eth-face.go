package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/faceuri"
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
	id := iface.FaceId(iface.FaceKind_Eth<<12) | iface.FaceId(port)
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
	return dpdk.EthDev(face.GetFaceId() & 0x0FFF)
}

func (face *EthFace) GetLocalUri() *faceuri.FaceUri {
	return faceuri.MustParse(fmt.Sprintf("ether://[%s]", face.GetPort().GetMacAddr()))
}

func (face *EthFace) GetRemoteUri() *faceuri.FaceUri {
	return faceuri.MustParse(fmt.Sprintf("dev://%s", face.GetPort().GetName()))
}

func (face *EthFace) Close() error {
	face.BeforeClose()
	C.EthFace_Close(face.getPtr())
	face.CloseBaseFace()
	return nil
}

func (face *EthFace) ReadExCounters() interface{} {
	return face.GetPort().GetStats()
}

func (face *EthFace) RxLoop(burstSize int, cb unsafe.Pointer, cbarg unsafe.Pointer) {
	C.EthFace_RxLoop(face.getPtr(), C.uint16_t(burstSize), (C.Face_RxCb)(cb), cbarg)
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
