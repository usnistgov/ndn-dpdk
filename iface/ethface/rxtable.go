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
	rxt  *rxTable
}

func newRxTableImpl(port *Port) impl {
	return &rxTableImpl{
		port: port,
	}
}

func (*rxTableImpl) String() string {
	return "RxTable"
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

func (impl *rxTableImpl) Start(face *ethFace) error {
	rxtC := impl.rxt.ptr()
	if face.loc.Remote.IsGroup() {
		return impl.setFace(&rxtC.multicast, face.ID())
	}
	lastOctet := face.loc.Remote.Bytes[5]
	return impl.setFace(&rxtC.unicast[lastOctet], face.ID())
}

func (impl *rxTableImpl) Stop(face *ethFace) error {
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

type rxTable C.EthRxTable

func newRxTable(port *Port) (rxt *rxTable) {
	c := (*C.EthRxTable)(eal.Zmalloc("EthRxTable", C.sizeof_EthRxTable, port.dev.NumaSocket()))
	c.port = C.uint16_t(port.dev.ID())
	c.queue = 0
	c.base.rxBurstOp = C.RxGroup_RxBurst(C.EthRxTable_RxBurst)
	c.base.rxThread = 0

	rxt = (*rxTable)(c)
	iface.EmitRxGroupAdd(rxt)
	return rxt
}

func (rxt *rxTable) ptr() *C.EthRxTable {
	return (*C.EthRxTable)(rxt)
}

func (*rxTable) IsRxGroup() {}

func (rxt *rxTable) NumaSocket() eal.NumaSocket {
	return ethdev.FromID(int(rxt.ptr().port)).NumaSocket()
}

func (rxt *rxTable) Ptr() unsafe.Pointer {
	return unsafe.Pointer(&rxt.ptr().base)
}

func (rxt *rxTable) Close() error {
	iface.EmitRxGroupRemove(rxt)
	eal.Free(rxt.ptr())
	return nil
}
