package ethface

/*
#include "../../csrc/ethface/rxtable.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go4.org/must"
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

func (impl *rxTableImpl) Start(face *ethFace) error {
	rxtC := impl.rxt.ptr()
	C.cds_hlist_add_head_rcu(&face.priv.rxtNode, &rxtC.head)
	return nil
}

func (impl *rxTableImpl) Stop(face *ethFace) error {
	C.cds_hlist_del_rcu(&face.priv.rxtNode)
	urcu.Barrier()
	return nil
}

func (impl *rxTableImpl) Close() error {
	if impl.rxt != nil {
		must.Close(impl.rxt)
		impl.rxt = nil
	}
	impl.port.dev.Stop(ethdev.StopReset)
	return nil
}

type rxTable C.EthRxTable

func newRxTable(port *Port) (rxt *rxTable) {
	socket := port.dev.NumaSocket()
	c := (*C.EthRxTable)(eal.Zmalloc("EthRxTable", C.sizeof_EthRxTable, socket))
	c.port = C.uint16_t(port.dev.ID())
	c.queue = 0
	c.base.rxBurstOp = C.RxGroup_RxBurst(C.EthRxTable_RxBurst)
	c.base.rxThread = 0
	if port.rxBouncePool != nil {
		rxPool := ndni.PacketMempool.Get(socket)
		c.copyTo = (*C.struct_rte_mempool)(rxPool.Ptr())
	}

	rxt = (*rxTable)(c)
	iface.ActivateRxGroup(rxt)
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
	iface.DeactivateRxGroup(rxt)
	eal.Free(rxt.ptr())
	return nil
}
