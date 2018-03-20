package mockface

/*
#include "mock-face.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type rxLoop struct{}

var theRxLoop rxLoop
var TheRxLoop iface.IRxLooper = &theRxLoop

type rxPacket struct {
	face *MockFace
	pkt  dpdk.Packet
}

var rxQueue chan rxPacket = make(chan rxPacket)
var rxStop chan struct{} = make(chan struct{})

func (rxl *rxLoop) RxLoop(burstSize int, cb unsafe.Pointer, cbarg unsafe.Pointer) {
	for {
		select {
		case rxp := <-rxQueue:
			C.MockFace_Rx(rxp.face.getPtr(), cb, cbarg, (*C.Packet)(rxp.pkt.GetPtr()))
		case <-rxStop:
			return
		}
	}
}

func (rxl *rxLoop) StopRxLoop() error {
	rxStop <- struct{}{}
	return nil
}

func (rxl *rxLoop) ListFacesInRxLoop() (faceIds []iface.FaceId) {
	faceIds = make([]iface.FaceId, 0)
	for id := minId; id <= maxId; id++ {
		if getById(id) != nil {
			faceIds = append(faceIds, iface.FaceId(id))
		}
	}
	return faceIds
}
