package ethport

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

type rxTable struct {
	rxt *rxgTable
}

func (rxTable) String() string {
	return "RxTable"
}

func (impl *rxTable) Init(port *Port) error {
	if e := port.startDev(1, true); e != nil {
		return e
	}
	impl.rxt = newRxgTable(port)
	return nil
}

func (impl *rxTable) Start(face *Face) error {
	C.cds_hlist_add_head_rcu(&face.priv.rxtNode, &impl.rxt.head)
	return nil
}

func (impl *rxTable) Stop(face *Face) error {
	C.cds_hlist_del_rcu(&face.priv.rxtNode)
	urcu.Barrier()
	return nil
}

func (impl *rxTable) Close(port *Port) error {
	if impl.rxt != nil {
		must.Close(impl.rxt)
		impl.rxt = nil
	}
	return nil
}

type rxgTable C.EthRxTable

var _ iface.RxGroup = &rxgTable{}

func (*rxgTable) IsRxGroup() {}

func (rxt *rxgTable) NumaSocket() eal.NumaSocket {
	return ethdev.FromID(int(rxt.port)).NumaSocket()
}

func (rxt *rxgTable) Ptr() unsafe.Pointer {
	return unsafe.Pointer(&rxt.base)
}

func (rxt *rxgTable) Close() error {
	iface.DeactivateRxGroup(rxt)
	eal.Free(rxt)
	return nil
}

func newRxgTable(port *Port) (rxt *rxgTable) {
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

	rxt = (*rxgTable)(c)
	iface.ActivateRxGroup(rxt)
	return rxt
}
