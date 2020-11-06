package ethface

/*
#include "../../csrc/ethface/face.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
)

const rxFlowMaxRxQueues = 4 // this may be set up to C.RTE_MAX_QUEUES_PER_PORT

// Read rte_flow_error into Go error.
func readFlowErr(e C.struct_rte_flow_error) error {
	if e._type == C.RTE_FLOW_ERROR_TYPE_NONE {
		return nil
	}
	return fmt.Errorf("%d %s %d", e._type, C.GoString(e.message), uintptr(e.cause))
}

type rxFlowImpl struct {
	port      *Port
	queueFlow []*rxFlow
}

func newRxFlowImpl(port *Port) impl {
	return &rxFlowImpl{
		port: port,
	}
}

func (*rxFlowImpl) String() string {
	return "RxFlow"
}

// Enter or leave flow isolation mode.
func (impl *rxFlowImpl) setIsolate(enable bool) error {
	var set C.int
	if enable {
		set = 1
	}
	var flowErr C.struct_rte_flow_error
	if res := C.rte_flow_isolate(C.uint16_t(impl.port.dev.ID()), set, &flowErr); res != 0 {
		return readFlowErr(flowErr)
	}
	return nil
}

func (impl *rxFlowImpl) Init() error {
	if impl.port.cfg.DisableRxFlow {
		return errors.New("disabled")
	}

	if e := impl.setIsolate(true); e != nil {
		return e
	}

	devInfo := impl.port.dev.DevInfo()
	nRxQueues := math.MinInt(int(devInfo.Max_rx_queues), rxFlowMaxRxQueues)
	if nRxQueues == 0 {
		return errors.New("unable to retrieve max_rx_queues")
	}

	if e := startDev(impl.port, nRxQueues, false); e != nil {
		return e
	}

	impl.queueFlow = make([]*rxFlow, nRxQueues)
	return nil
}

func (impl *rxFlowImpl) findQueue(filter func(rxf *rxFlow) bool) (i int, rxf *rxFlow) {
	for i, rxf = range impl.queueFlow {
		if filter(rxf) {
			return
		}
	}
	return -1, nil
}

func (impl *rxFlowImpl) Start(face *ethFace) error {
	index, _ := impl.findQueue(func(rxf *rxFlow) bool { return rxf == nil })
	if index < 0 {
		// TODO reclaim deferred-destroy queues
		return errors.New("no available queue")
	}

	rxf, e := newRxFlow(face, index)
	if e != nil {
		return e
	}

	impl.port.logger.WithFields(makeLogFields("rx-queue", index, "face", face.ID())).Debug("create RxFlow")
	impl.queueFlow[index] = rxf
	iface.EmitRxGroupAdd(rxf)
	return nil
}

func (impl *rxFlowImpl) Stop(face *ethFace) error {
	index, rxf := impl.findQueue(func(rxf *rxFlow) bool { return rxf != nil && rxf.face == face })
	if index < 0 {
		return nil
	}
	iface.EmitRxGroupRemove(rxf)

	if e := impl.destroyFlow(rxf); e != nil {
		impl.port.logger.WithField("rx-queue", index).WithError(e).Debug("destroy RxFlow deferred")
		rxf.face = nil
	} else {
		impl.port.logger.WithField("rx-queue", index).Debug("destroy RxFlow success")
		impl.queueFlow[index] = nil
	}
	return nil
}

func (impl *rxFlowImpl) destroyFlow(rxf *rxFlow) error {
	var flowErr C.struct_rte_flow_error
	if res := C.rte_flow_destroy(C.uint16_t(impl.port.dev.ID()), rxf.flow, &flowErr); res != 0 {
		return readFlowErr(flowErr)
	}
	return nil
}

func (impl *rxFlowImpl) Close() error {
	for _, rxf := range impl.queueFlow {
		if rxf != nil {
			impl.destroyFlow(rxf)
		}
	}
	impl.queueFlow = nil
	impl.port.dev.Stop(ethdev.StopReset)
	impl.setIsolate(false)
	return nil
}

type rxFlow struct {
	face *ethFace
	flow *C.struct_rte_flow
}

func newRxFlow(face *ethFace, queue int) (*rxFlow, error) {
	priv := face.priv
	priv.rxQueue = C.uint16_t(queue)

	cLoc := face.loc.cLoc()
	var flowErr C.struct_rte_flow_error
	flow := C.EthFace_SetupFlow(priv, cLoc.ptr(), &flowErr)
	if flow == nil {
		return nil, readFlowErr(flowErr)
	}

	rxf := &rxFlow{
		face: face,
		flow: flow,
	}
	priv.flowRxg.rxBurstOp = C.RxGroup_RxBurst(C.EthFace_FlowRxBurst)
	priv.flowRxg.rxThread = 0
	return rxf, nil
}

func (*rxFlow) IsRxGroup() {}

func (rxf *rxFlow) NumaSocket() eal.NumaSocket {
	return rxf.face.NumaSocket()
}

func (rxf *rxFlow) Ptr() unsafe.Pointer {
	return unsafe.Pointer(&rxf.face.priv.flowRxg)
}
