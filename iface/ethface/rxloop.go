package ethface

/*
#include "rxloop.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type RxLoop struct {
	c       *C.EthRxLoop
	stopped chan bool
}

func NewRxLoop(maxTasks int, numaSocket dpdk.NumaSocket) (rxl *RxLoop) {
	rxl = new(RxLoop)
	rxl.c = C.EthRxLoop_New(C.int(maxTasks), C.int(numaSocket))
	if rxl.c == nil {
		panic("out of memory")
	}
	rxl.stopped = make(chan bool)
	return rxl
}

func (rxl *RxLoop) Close() error {
	C.EthRxLoop_Close(rxl.c)
	return nil
}

func (rxl *RxLoop) Add(face *EthFace) error {
	if rxl.c.nTasks >= rxl.c.maxTasks {
		return fmt.Errorf("this RxLoop is full")
	}

	if face.nRxThreads >= C.RXPROC_MAX_THREADS {
		return fmt.Errorf("cannot add face to more than %d RxLoops", C.RXPROC_MAX_THREADS)
	}
	if ethDevInfo := face.GetPort().GetDevInfo(); face.nRxThreads >= int(ethDevInfo.Nb_rx_queues) {
		return fmt.Errorf("cannot add this face to more than %d RxLoops", ethDevInfo.Nb_rx_queues)
	}

	var task C.EthRxTask
	task.port = C.uint16_t(face.GetPort())
	task.queue = C.uint16_t(face.nRxThreads)
	task.rxThread = C.int(face.nRxThreads)
	task.face = face.getPtr()
	C.EthRxLoop_AddTask(rxl.c, &task)

	face.nRxThreads++
	return nil
}

func (rxl *RxLoop) RxLoop(burstSize int, cb unsafe.Pointer, cbarg unsafe.Pointer) {
	burst := iface.NewRxBurst(burstSize)
	defer burst.Close()
	C.EthRxLoop_Run(rxl.c, (*C.FaceRxBurst)(burst.GetPtr()), (C.Face_RxCb)(cb), cbarg)
	rxl.stopped <- true
}

func (rxl *RxLoop) StopRxLoop() error {
	rxl.c.stop = true
	<-rxl.stopped
	rxl.c.stop = false
	return nil
}

func (rxl *RxLoop) ListFacesInRxLoop() []iface.FaceId {
	list := make([]iface.FaceId, int(rxl.c.nTasks))
	for i := range list {
		taskC := C.__EthRxLoop_GetTask(rxl.c, C.int(i))
		list[i] = iface.FaceId(taskC.face.id)
	}
	return list
}
