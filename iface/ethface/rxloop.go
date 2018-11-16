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

func (rxl *RxLoop) GetNumaSocket() dpdk.NumaSocket {
	if rxl.c.nTasks == 0 {
		return dpdk.NUMA_SOCKET_ANY
	}
	taskC := C.__EthRxLoop_GetTask(rxl.c, 0)
	port := FindPort(dpdk.EthDev(taskC.port))
	return port.GetNumaSocket()
}

func (rxl *RxLoop) AddPort(port *Port) error {
	if rxl.c.nTasks >= rxl.c.maxTasks {
		return fmt.Errorf("this RxLoop is full")
	}

	if port.nRxThreads >= C.RXPROC_MAX_THREADS {
		return fmt.Errorf("cannot add port to more than %d RxLoops", C.RXPROC_MAX_THREADS)
	}
	if di := port.dev.GetDevInfo(); port.nRxThreads >= int(di.Nb_rx_queues) {
		return fmt.Errorf("cannot add this face to more than %d RxLoops", di.Nb_rx_queues)
	}

	var taskC C.EthRxTask
	taskC.port = C.uint16_t(port.dev)
	taskC.queue = C.uint16_t(port.nRxThreads)
	taskC.rxThread = C.int(port.nRxThreads)
	if port.multicast != nil {
		taskC.multicast = C.FaceId(port.multicast.GetFaceId())
	}
	for _, face := range port.unicast {
		taskC.unicast[face.remote[5]] = C.FaceId(face.GetFaceId())
	}
	C.EthRxLoop_AddTask(rxl.c, &taskC)

	port.nRxThreads++
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

func (rxl *RxLoop) ListFacesInRxLoop() (list []iface.FaceId) {
	for i := C.int(0); i < rxl.c.nTasks; i++ {
		taskC := C.__EthRxLoop_GetTask(rxl.c, C.int(i))
		if taskC.multicast != 0 {
			list = append(list, iface.FaceId(taskC.multicast))
		}
		for j := 0; j < 256; j++ {
			if taskC.unicast[j] != 0 {
				list = append(list, iface.FaceId(taskC.unicast[j]))
			}
		}
	}
	return list
}
