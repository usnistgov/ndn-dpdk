package mockface

/*
#include "../face.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type rxLoop struct{}

var TheRxLoop iface.IRxLooper = rxLoop{}

var rxQueue chan dpdk.Packet = make(chan dpdk.Packet)
var rxStop chan struct{} = make(chan struct{})

func (rxLoop) RxLoop(burstSize int, cb unsafe.Pointer, cbarg unsafe.Pointer) {
	burst := iface.NewRxBurst(1)
	defer burst.Close()
	for {
		select {
		case pkt := <-rxQueue:
			burst.SetFrame(0, pkt)
			C.FaceImpl_RxBurst((*C.FaceRxBurst)(burst.GetPtr()), 1, 0, (C.Face_RxCb)(cb), cbarg)
		case <-rxStop:
			return
		}
	}
}

func (rxLoop) StopRxLoop() error {
	rxStop <- struct{}{}
	return nil
}

func (rxLoop) ListFacesInRxLoop() (faceIds []iface.FaceId) {
	faceIds = make([]iface.FaceId, 0)
	for it := iface.IterFaces(); it.Valid(); it.Next() {
		if it.Id.GetKind() == iface.FaceKind_Mock {
			faceIds = append(faceIds, it.Id)
		}
	}
	return faceIds
}
