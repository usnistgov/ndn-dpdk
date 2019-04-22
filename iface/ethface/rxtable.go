package ethface

/*
#include "rxgroup.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type rxTableStarter struct{}

func (rxTableStarter) String() string {
	return "RxTable"
}

// Start port with software-dispatched RxTable.
func (rxTableStarter) Start(port *Port, cfg PortConfig) error {
	unicastByLastOctet := make(map[byte]int)
	for i, addr := range cfg.Unicast {
		if j, ok := unicastByLastOctet[addr[5]]; ok {
			return fmt.Errorf("cfg.Unicast[%d] has same last octet with cfg.Unicast[%d]", i, j)
		}
		unicastByLastOctet[addr[5]] = i
	}

	if e := port.configureDev(cfg, cfg.NRxThreads); e != nil {
		return e
	}
	port.dev.SetPromiscuous(true)
	if e := port.startDev(); e != nil {
		return e
	}

	port.createFaces(cfg, nil)
	for i := 0; i < cfg.NRxThreads; i++ {
		port.rxt = append(port.rxt, newRxTable(port, i))
	}
	return nil
}

// Table-based software RX dispatching.
type RxTable struct {
	iface.RxGroupBase
	c    *C.EthRxTable
	port *Port
}

func newRxTable(port *Port, rxThreadId int) (rxt *RxTable) {
	rxt = new(RxTable)
	rxt.c = (*C.EthRxTable)(dpdk.Zmalloc("EthRxTable", C.sizeof_EthRxTable, port.GetNumaSocket()))
	rxt.InitRxgBase(unsafe.Pointer(rxt.c))
	rxt.port = port

	rxt.c.port = C.uint16_t(port.dev)
	rxt.c.queue = C.uint16_t(rxThreadId)
	rxt.c.base.rxBurstOp = C.RxGroup_RxBurst(C.EthRxTable_RxBurst)
	rxt.c.base.rxThread = C.int(rxThreadId)

	if port.multicast != nil {
		rxt.c.multicast = C.FaceId(port.multicast.GetFaceId())
	}
	for _, face := range port.unicast {
		rxt.c.unicast[face.remote[5]] = C.FaceId(face.GetFaceId())
	}

	return rxt
}

func (rxt *RxTable) Close() error {
	dpdk.Free(rxt.c)
	return nil
}

func (rxt *RxTable) GetNumaSocket() dpdk.NumaSocket {
	return rxt.port.GetNumaSocket()
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
