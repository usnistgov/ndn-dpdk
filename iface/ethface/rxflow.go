package ethface

/*
#include "rxgroup.h"
*/
import "C"
import (
	"errors"
	"net"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

var DisableRxFlow = false

// Read rte_flow_error into Go error.
func readFlowErr(flowErr C.struct_rte_flow_error) error {
	if flowErr._type == C.RTE_FLOW_ERROR_TYPE_NONE {
		return nil
	}
	return errors.New(C.GoString(flowErr.message))
}

type rxFlowStarter struct{}

func (rxFlowStarter) String() string {
	return "RxFlow"
}

// Start port with hardware-dispatched RxFlows.
func (rxFlowStarter) Start(port *Port, cfg PortConfig) (e error) {
	if DisableRxFlow {
		return errors.New("RxFlow disabled")
	}

	var flowErr C.struct_rte_flow_error
	if res := C.rte_flow_isolate(C.uint16_t(port.dev), 1, &flowErr); res != 0 {
		return readFlowErr(flowErr)
	}

	flows := make(map[iface.FaceId]*RxFlow)
	defer func() {
		if e == nil {
			return
		}
		for _, flow := range flows {
			flow.Close()
		}
		C.rte_flow_isolate(C.uint16_t(port.dev), 0, &flowErr)
	}()

	nFaces := cfg.countFaces()
	if e = port.configureDev(cfg, nFaces); e != nil {
		return e
	}

	for i := 0; i < nFaces; i++ {
		id, addr := cfg.getFaceIdAddr(i)
		flow := newRxFlow(port, i, id)
		flows[id] = flow
		if e = flow.setup(addr); e != nil {
			return e
		}
	}

	if e = port.startDev(); e != nil {
		return e
	}

	port.createFaces(cfg, flows)
	return nil
}

// rte_flow-based hardware RX dispatching.
type RxFlow struct {
	iface.RxGroupBase
	c    *C.EthRxFlow
	port *Port
}

func newRxFlow(port *Port, queue int, face iface.FaceId) (rxf *RxFlow) {
	rxf = new(RxFlow)
	rxf.c = (*C.EthRxFlow)(dpdk.Zmalloc("EthRxFlow", C.sizeof_EthRxFlow, port.GetNumaSocket()))
	rxf.InitRxgBase(unsafe.Pointer(rxf.c))
	rxf.port = port

	rxf.c.port = C.uint16_t(port.dev)
	rxf.c.queue = C.uint16_t(queue)
	rxf.c.base.rxBurstOp = C.RxGroup_RxBurst(C.EthRxFlow_RxBurst)
	rxf.c.base.rxThread = 0

	rxf.c.face = C.FaceId(face)
	return rxf
}

func (rxf *RxFlow) setup(addr net.HardwareAddr) error {
	var addrC C.struct_ether_addr
	var addrCP *C.struct_ether_addr
	if addr != nil {
		addrCP = &addrC
		copyHwaddrToC(addr, addrCP)
	}

	var flowErr C.struct_rte_flow_error
	C.EthRxFlow_Setup(rxf.c, addrCP, &flowErr)
	return readFlowErr(flowErr)
}

func (rxf *RxFlow) Close() error {
	if rxf.c.flow != nil {
		var flowErr C.struct_rte_flow_error
		C.rte_flow_destroy(C.uint16_t(rxf.port.dev), rxf.c.flow, &flowErr)
	}
	dpdk.Free(rxf.c)
	return nil
}

func (rxf *RxFlow) GetNumaSocket() dpdk.NumaSocket {
	return rxf.port.GetNumaSocket()
}

func (rxf *RxFlow) ListFaces() []iface.FaceId {
	return []iface.FaceId{iface.FaceId(rxf.c.face)}
}
