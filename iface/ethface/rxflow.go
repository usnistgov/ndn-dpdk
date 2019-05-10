package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"errors"
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

type rxFlowImpl struct {
	port      *Port
	queueFlow []*RxFlow
}

func newRxFlowImpl(port *Port) (impl *rxFlowImpl) {
	impl = new(rxFlowImpl)
	impl.port = port
	return impl
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
	if res := C.rte_flow_isolate(C.uint16_t(impl.port.dev), set, &flowErr); res != 0 {
		return readFlowErr(flowErr)
	}
	return nil
}

func (impl *rxFlowImpl) Init() error {
	if DisableRxFlow {
		return errors.New("disabled")
	}

	if e := impl.setIsolate(true); e != nil {
		return e
	}

	devInfo := impl.port.dev.GetDevInfo()
	nRxQueues := int(devInfo.Max_rx_queues)
	if nRxQueues == 0 {
		return errors.New("unable to retrieve max_rx_queues")
	}
	if nRxQueues > C.RTE_MAX_QUEUES_PER_PORT {
		nRxQueues = C.RTE_MAX_QUEUES_PER_PORT
	}
	nRxQueues = 4

	if e := impl.port.startDev(nRxQueues, false); e != nil {
		return e
	}

	impl.queueFlow = make([]*RxFlow, nRxQueues)
	return nil
}

func (impl *rxFlowImpl) findQueue(filter func(rxf *RxFlow) bool) (i int, rxf *RxFlow) {
	for i, rxf = range impl.queueFlow {
		if filter(rxf) {
			return
		}
	}
	return -1, nil
}

func (impl *rxFlowImpl) Start(face *EthFace) error {
	index, _ := impl.findQueue(func(rxf *RxFlow) bool { return rxf == nil })
	if index < 0 {
		// TODO reclaim deferred-destroy queues
		return errors.New("no available queue")
	}

	rxf, e := newRxFlow(face, index)
	if e != nil {
		return e
	}

	impl.port.logger.WithFields(makeLogFields("rx-queue", index, "face", face.GetFaceId())).Debug("create RxFlow")
	impl.queueFlow[index] = rxf
	return nil
}

func (impl *rxFlowImpl) Stop(face *EthFace) error {
	index, rxf := impl.findQueue(func(rxf *RxFlow) bool { return rxf != nil && rxf.face == face })
	if index < 0 {
		return nil
	}

	if e := impl.destroyFlow(rxf); e != nil {
		impl.port.logger.WithField("rx-queue", index).WithError(e).Debug("destroy RxFlow deferred")
		rxf.face = nil
	} else {
		impl.port.logger.WithField("rx-queue", index).Debug("destroy RxFlow success")
		impl.queueFlow[index] = nil
	}
	return nil
}

func (impl *rxFlowImpl) destroyFlow(rxf *RxFlow) error {
	var flowErr C.struct_rte_flow_error
	if res := C.rte_flow_destroy(C.uint16_t(impl.port.dev), rxf.flow, &flowErr); res != 0 {
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
	impl.port.dev.Stop()
	impl.setIsolate(false)
	return nil
}

// rte_flow-based hardware RX dispatching.
type RxFlow struct {
	iface.RxGroupBase
	face *EthFace
	flow *C.struct_rte_flow
}

func newRxFlow(face *EthFace, queue int) (rxf *RxFlow, e error) {
	priv := face.getPriv()
	priv.rxQueue = C.uint16_t(queue)
	var flowErr C.struct_rte_flow_error
	flow := C.EthFace_SetupFlow(priv, &flowErr)
	if flow == nil {
		return nil, readFlowErr(flowErr)
	}

	rxf = new(RxFlow)
	rxf.InitRxgBase(unsafe.Pointer(&priv.flowRxg))
	rxf.face = face
	rxf.flow = flow
	priv.flowRxg.rxBurstOp = C.RxGroup_RxBurst(C.EthFace_FlowRxBurst)
	priv.flowRxg.rxThread = 0
	return rxf, nil
}

func (rxf *RxFlow) Close() error {
	return nil
}

func (rxf *RxFlow) GetNumaSocket() dpdk.NumaSocket {
	return rxf.face.GetNumaSocket()
}

func (rxf *RxFlow) ListFaces() []iface.FaceId {
	return []iface.FaceId{rxf.face.GetFaceId()}
}
