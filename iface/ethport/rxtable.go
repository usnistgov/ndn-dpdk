package ethport

/*
#include "../../csrc/ethface/rxtable.h"
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/zap"
	"go4.org/must"
)

type rxTable struct {
	rxt       *rxgTable
	flowFlags C.EthFlowFlags
}

func (rxTable) String() string {
	return "RxTable"
}

func (impl *rxTable) List(port *Port) []iface.RxGroup {
	return []iface.RxGroup{impl.rxt}
}

func (impl *rxTable) Init(port *Port) error {
	if e := port.startDev(1, true); e != nil {
		return e
	}
	impl.rxt = newRxgTable(port)
	impl.flowFlags = C.EthFlowFlags(port.devInfo.FlowFlags())
	return nil
}

func (impl *rxTable) Start(face *Face) error {
	if impl.flowFlags&C.EthFlowFlagsDisabled == 0 {
		setupFlow(face, []uint16{0}, impl.flowFlags, zap.InfoLevel)
	}

	if face.loc.Scheme() == SchemePassthru {
		C.cds_list_add_tail_rcu(&face.priv.rxtNode, &impl.rxt.head)
	} else {
		C.cds_list_add_rcu(&face.priv.rxtNode, &impl.rxt.head)
	}
	return nil
}

func (impl *rxTable) Stop(face *Face) error {
	C.cds_list_del_rcu(&face.priv.rxtNode)
	urcu.Synchronize()

	destroyFlow(face)
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

func (rxt *rxgTable) ethDev() ethdev.EthDev {
	return ethdev.FromID(int(rxt.port))
}

func (rxt *rxgTable) NumaSocket() eal.NumaSocket {
	return rxt.ethDev().NumaSocket()
}

func (rxt *rxgTable) RxGroup() (ptr unsafe.Pointer, desc string) {
	return unsafe.Pointer(&rxt.base),
		fmt.Sprintf("EthRxTable(port=%d,queue=%d)", rxt.port, rxt.queue)
}

func (rxt *rxgTable) Faces() []iface.Face {
	port := Find(rxt.ethDev())
	return port.Faces()
}

func (rxt *rxgTable) Close() error {
	iface.DeactivateRxGroup(rxt)
	eal.Free(rxt)
	return nil
}

func newRxgTable(port *Port) (rxt *rxgTable) {
	socket := port.dev.NumaSocket()
	rxt = eal.Zmalloc[rxgTable]("EthRxTable", C.sizeof_EthRxTable, socket)
	C.EthRxTable_Init((*C.EthRxTable)(rxt), C.uint16_t(port.dev.ID()))
	if port.rxBouncePool != nil {
		rxPool := ndni.PacketMempool.Get(socket)
		rxt.copyTo = (*C.struct_rte_mempool)(rxPool.Ptr())
	}

	iface.ActivateRxGroup(rxt)
	return rxt
}
