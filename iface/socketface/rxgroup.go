package socketface

/*
#include "../face.h"
*/
import "C"
import (
	"errors"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

// Provide RxLoop for a set of SocketFaces.
type RxGroup struct {
	lock        sync.Mutex
	quit        chan<- bool
	ticker      *time.Ticker
	faces       []*SocketFace
	selectCases []reflect.SelectCase
}

func NewRxGroup(faces ...*SocketFace) *RxGroup {
	var rxg RxGroup
	quit := make(chan bool)
	rxg.quit = quit
	rxg.ticker = time.NewTicker(10 * time.Millisecond)
	rxg.selectCases = []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(quit)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(rxg.ticker.C)},
	}

	for _, face := range faces {
		rxg.AddFace(face)
	}
	return &rxg
}

func (rxg *RxGroup) Close() error {
	return rxg.StopRxLoop()
}

func (rxg *RxGroup) AddFace(face *SocketFace) error {
	rxg.lock.Lock()
	defer rxg.lock.Unlock()

	for _, f := range rxg.faces {
		if f == face {
			return errors.New("face is already in RxGroup")
		}
	}

	rxg.faces = append(rxg.faces, face)
	rxg.selectCases = append(rxg.selectCases,
		reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(face.rxQueue)})
	return nil
}

func (rxg *RxGroup) RemoveFace(face *SocketFace) error {
	rxg.lock.Lock()
	defer rxg.lock.Unlock()

	index := -1
	for i, f := range rxg.faces {
		if f == face {
			index = i
			break
		}
	}
	if index < 0 {
		return errors.New("face is not in RxGroup")
	}

	last := len(rxg.faces) - 1
	rxg.faces[index] = rxg.faces[last]
	rxg.faces = rxg.faces[:last]
	rxg.selectCases[index+2] = rxg.selectCases[last+2]
	rxg.selectCases = rxg.selectCases[:last+2]
	return nil
}

func (rxg *RxGroup) RxLoop(burstSize int, cb unsafe.Pointer, cbarg unsafe.Pointer) {
	burst := iface.NewRxBurst(burstSize)
	defer burst.Close()
	stop := false
	for !stop {
		rxg.lock.Lock()
		chosen, recv, _ := reflect.Select(rxg.selectCases)
		if chosen <= 1 { // 0=quit, 1=ticker for periodical unlock
			stop = chosen == 0
		} else { // RX
			face := rxg.faces[chosen-2]
			nRx := 1
			burst.SetFrame(0, recv.Interface().(dpdk.Packet))
		LOOP_BURST:
			for ; nRx < burstSize; nRx++ {
				select {
				case pkt := <-face.rxQueue:
					burst.SetFrame(nRx, pkt)
				default:
					break LOOP_BURST
				}
			}
			C.FaceImpl_RxBurst(face.getPtr(), (*C.FaceRxBurst)(burst.GetPtr()),
				C.uint16_t(nRx), (C.Face_RxCb)(cb), cbarg)
		}
		rxg.lock.Unlock()
	}
}

func (rxg *RxGroup) StopRxLoop() error {
	rxg.quit <- true
	return nil
}

func (rxg *RxGroup) ListFacesInRxLoop() []iface.FaceId {
	rxg.lock.Lock()
	defer rxg.lock.Unlock()

	list := make([]iface.FaceId, len(rxg.faces))
	for i, face := range rxg.faces {
		list[i] = face.GetFaceId()
	}
	return list
}
