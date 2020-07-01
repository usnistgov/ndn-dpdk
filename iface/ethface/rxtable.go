package ethface

/*
#include "../../csrc/ethface/rxtable.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
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

func (impl *rxTableImpl) setFace(slot *C.FaceID, faceID iface.ID) error {
	oldFaceID := iface.ID(*slot)
	if impl.port.faces[oldFaceID] != nil {
		return fmt.Errorf("new face %d conflicts with old face %d", faceID, oldFaceID)
	}
	*slot = C.FaceID(faceID)
	return nil
}

func (impl *rxTableImpl) Start(face *EthFace) error {
	if face.loc.Remote.IsGroup() {
		return impl.setFace(&impl.rxt.c.multicast, face.ID())
	}
	lastOctet := face.loc.Remote.Bytes[5]
	return impl.setFace(&impl.rxt.c.unicast[lastOctet], face.ID())
}

func (impl *rxTableImpl) Stop(face *EthFace) error {
	return nil
}

func (impl *rxTableImpl) Close() error {
	if impl.rxt != nil {
		impl.rxt.Close()
		impl.rxt = nil
	}
	impl.port.dev.Stop(ethdev.StopReset)
	return nil
}

// Table-based software RX dispatching.
type RxTable struct {
	iface.RxGroupBase
	c *C.EthRxTable
}

func newRxTable(port *Port) (rxt *RxTable) {
	rxt = new(RxTable)
	rxt.c = (*C.EthRxTable)(eal.Zmalloc("EthRxTable", C.sizeof_EthRxTable, port.dev.NumaSocket()))
	rxt.InitRxgBase(unsafe.Pointer(rxt.c))

	rxt.c.port = C.uint16_t(port.dev.ID())
	rxt.c.queue = 0
	rxt.c.base.rxBurstOp = C.RxGroup_RxBurst(C.EthRxTable_RxBurst)
	rxt.c.base.rxThread = 0

	iface.EmitRxGroupAdd(rxt)
	return rxt
}

func (rxt *RxTable) Close() error {
	iface.EmitRxGroupRemove(rxt)
	eal.Free(rxt.c)
	return nil
}

func (rxt *RxTable) NumaSocket() eal.NumaSocket {
	return ethdev.FromID(int(rxt.c.port)).NumaSocket()
}

func (rxt *RxTable) ListFaces() (list []iface.ID) {
	if rxt.c.multicast != 0 {
		list = append(list, iface.ID(rxt.c.multicast))
	}
	for j := 0; j < 256; j++ {
		if rxt.c.unicast[j] != 0 {
			list = append(list, iface.ID(rxt.c.unicast[j]))
		}
	}
	return list
}
