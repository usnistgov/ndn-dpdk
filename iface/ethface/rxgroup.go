package ethface

/*
#include "rxgroup.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type RxGroup struct {
	iface.RxGroupBase
	c    *C.EthRxGroup
	port *Port
}

func newRxGroup(port *Port, queue, rxThread int) (rxg *RxGroup) {
	rxg = new(RxGroup)
	rxg.c = (*C.EthRxGroup)(dpdk.Zmalloc("EthRxGroup", C.sizeof_EthRxGroup, port.GetNumaSocket()))
	rxg.InitRxgBase(unsafe.Pointer(rxg.c))
	rxg.port = port

	rxg.c.port = C.uint16_t(port.dev)
	rxg.c.queue = C.uint16_t(queue)
	rxg.c.base.rxBurstOp = C.RxGroup_RxBurst(C.EthRxGroup_RxBurst)
	rxg.c.base.rxThread = C.int(rxThread)

	if port.multicast != nil {
		rxg.c.multicast = C.FaceId(port.multicast.GetFaceId())
	}
	for _, face := range port.unicast {
		rxg.c.unicast[face.remote[5]] = C.FaceId(face.GetFaceId())
	}

	return rxg
}

func (rxg *RxGroup) Close() error {
	dpdk.Free(rxg.c)
	return nil
}

func (rxg *RxGroup) GetNumaSocket() dpdk.NumaSocket {
	return rxg.port.GetNumaSocket()
}

func (rxg *RxGroup) ListFaces() (list []iface.FaceId) {
	if rxg.c.multicast != 0 {
		list = append(list, iface.FaceId(rxg.c.multicast))
	}
	for j := 0; j < 256; j++ {
		if rxg.c.unicast[j] != 0 {
			list = append(list, iface.FaceId(rxg.c.unicast[j]))
		}
	}
	return list
}
