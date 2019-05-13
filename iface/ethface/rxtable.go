package ethface

/*
#include "rxtable.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type rxTableImpl struct {
	port *Port
	rxt  *RxTable
}

func (*rxTableImpl) String() string {
	return "RxTable"
}

func (*rxTableImpl) New(port *Port) iImpl {
	impl := new(rxTableImpl)
	impl.port = port
	return impl
}

func (impl *rxTableImpl) Init() error {
	if e := startDev(impl.port, 1, true); e != nil {
		return e
	}
	impl.rxt = newRxTable(impl.port)
	return nil
}

func (impl *rxTableImpl) setFace(slot *C.FaceId, faceId iface.FaceId) error {
	oldFaceId := iface.FaceId(*slot)
	if impl.port.faces[oldFaceId] != nil {
		return fmt.Errorf("new face %d conflicts with old face %d", faceId, oldFaceId)
	}
	*slot = C.FaceId(faceId)
	return nil
}

func (impl *rxTableImpl) Start(face *EthFace) error {
	if classifyMac48(face.remote) == mac48_multicast {
		return impl.setFace(&impl.rxt.c.multicast, face.GetFaceId())
	}
	lastOctet := face.remote[5]
	return impl.setFace(&impl.rxt.c.unicast[lastOctet], face.GetFaceId())
}

func (impl *rxTableImpl) Stop(face *EthFace) error {
	return nil
}

func (impl *rxTableImpl) Close() error {
	if impl.rxt != nil {
		impl.rxt.Close()
		impl.rxt = nil
	}
	impl.port.dev.Stop()
	return nil
}

// Table-based software RX dispatching.
type RxTable struct {
	iface.RxGroupBase
	c *C.EthRxTable
}

func newRxTable(port *Port) (rxt *RxTable) {
	rxt = new(RxTable)
	rxt.c = (*C.EthRxTable)(dpdk.Zmalloc("EthRxTable", C.sizeof_EthRxTable, port.dev.GetNumaSocket()))
	rxt.InitRxgBase(unsafe.Pointer(rxt.c))

	rxt.c.port = C.uint16_t(port.dev)
	rxt.c.queue = 0
	rxt.c.base.rxBurstOp = C.RxGroup_RxBurst(C.EthRxTable_RxBurst)
	rxt.c.base.rxThread = 0

	iface.EmitRxGroupAdd(rxt)
	return rxt
}

func (rxt *RxTable) Close() error {
	iface.EmitRxGroupRemove(rxt)
	dpdk.Free(rxt.c)
	return nil
}

func (rxt *RxTable) GetNumaSocket() dpdk.NumaSocket {
	return dpdk.EthDev(rxt.c.port).GetNumaSocket()
}

func (rxt *RxTable) ListFaces() (list []iface.FaceId) {
	if rxt.c.multicast != 0 {
		list = append(list, iface.FaceId(rxt.c.multicast))
	}
	for j := 0; j < 256; j++ {
		if rxt.c.unicast[j] != 0 {
			list = append(list, iface.FaceId(rxt.c.unicast[j]))
		}
	}
	return list
}
