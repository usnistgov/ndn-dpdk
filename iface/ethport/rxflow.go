package ethport

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
	"go.uber.org/zap"
)

type rxqState int

const (
	rxqStateIdle rxqState = iota
	rxqStateInUse
	rxqStateDeferred
)

// Read rte_flow_error into Go error.
func readFlowErr(e C.struct_rte_flow_error) error {
	if e._type == C.RTE_FLOW_ERROR_TYPE_NONE {
		return nil
	}

	causeFmt, cause := "%p", interface{}(e.cause)
	if causeOffset := uintptr(e.cause); causeOffset < uintptr(C.sizeof_EthFlowPattern) {
		causeFmt, cause = "%d", causeOffset
	}

	return fmt.Errorf("%d %s "+causeFmt, e._type, C.GoString(e.message), cause)
}

type rxFlow struct {
	isolated bool
	queues   []rxqState
}

func (rxFlow) String() string {
	return "RxFlow"
}

// setIsolate enters or leaves flow isolation mode.
func (impl *rxFlow) setIsolate(port *Port, enable bool) error {
	var set C.int
	if enable {
		set = 1
	}

	var flowErr C.struct_rte_flow_error
	if res := C.rte_flow_isolate(C.uint16_t(port.dev.ID()), set, &flowErr); res != 0 {
		return readFlowErr(flowErr)
	}
	impl.isolated = enable
	return nil
}

func (impl *rxFlow) Init(port *Port) error {
	if e := impl.setIsolate(port, true); e != nil {
		port.logger.Info("flow isolated mode unavailable", zap.Error(e))
	}

	maxRxQueues := int(port.devInfo.Max_rx_queues)
	if port.cfg.RxFlowQueues > maxRxQueues {
		return fmt.Errorf("%d RX queues requested but only %d available", port.cfg.RxFlowQueues, maxRxQueues)
	}

	if e := port.startDev(port.cfg.RxFlowQueues, !impl.isolated); e != nil {
		return e
	}

	impl.queues = make([]rxqState, port.cfg.RxFlowQueues)
	return nil
}

func (impl *rxFlow) Start(face *Face) error {
	queues := impl.allocQueues(math.MinInt(math.MaxInt(1, face.loc.EthFaceConfig().MaxRxQueues), iface.MaxRxProcThreads))
	if len(queues) == 0 {
		// TODO reclaim deferred-destroy queues
		return errors.New("no available queue")
	}

	if e := impl.setupFlow(face, queues); e != nil {
		return e
	}

	face.port.logger.Debug("create RxFlow",
		zap.Ints("queues", queues),
		face.ID().ZapField("face"),
	)
	impl.startFlow(face, queues)
	return nil
}

func (impl *rxFlow) allocQueues(max int) (queues []int) {
	for rxq, state := range impl.queues {
		if len(queues) < max && state == rxqStateIdle {
			queues = append(queues, rxq)
		}
	}
	return
}

func (impl *rxFlow) setupFlow(face *Face, queues []int) error {
	var cQueues []C.uint16_t
	for _, q := range queues {
		cQueues = append(cQueues, C.uint16_t(q))
	}

	cLoc := face.loc.EthCLocator()
	var flowErr C.struct_rte_flow_error
	face.flow = C.EthFace_SetupFlow(face.priv, &cQueues[0], C.int(len(queues)), cLoc.ptr(), C.bool(impl.isolated), &flowErr)
	if face.flow == nil {
		return readFlowErr(flowErr)
	}
	return nil
}

func (impl *rxFlow) startFlow(face *Face, queues []int) {
	face.rxf = make([]*rxgFlow, len(queues))
	for i, queue := range queues {
		impl.queues[queue] = rxqStateInUse
		rxf := &rxgFlow{
			face:  face,
			index: i,
			queue: queue,
		}
		face.rxf[i] = rxf
		iface.ActivateRxGroup(rxf)
	}
}

func (impl *rxFlow) Stop(face *Face) error {
	if face.flow == nil {
		return nil
	}

	for _, rxf := range face.rxf {
		iface.DeactivateRxGroup(rxf)
	}

	nextState := rxqStateIdle
	if e := impl.destroyFlow(face); e != nil {
		face.port.logger.Debug("destroy RxFlow deferred",
			face.ID().ZapField("face"),
			zap.Error(e),
		)
		nextState = rxqStateDeferred
	} else {
		face.port.logger.Debug("destroy RxFlow success",
			face.ID().ZapField("face"),
		)
	}

	for _, rxf := range face.rxf {
		impl.queues[rxf.queue] = nextState
	}
	face.rxf = nil
	return nil
}

func (impl *rxFlow) destroyFlow(face *Face) error {
	if face.flow == nil {
		return nil
	}

	var e C.struct_rte_flow_error
	if res := C.rte_flow_destroy(C.uint16_t(face.port.dev.ID()), face.flow, &e); res != 0 {
		return readFlowErr(e)
	}
	face.flow = nil
	return nil
}

func (impl *rxFlow) Close(port *Port) error {
	for _, face := range port.faces {
		impl.destroyFlow(face)
	}
	impl.queues = nil

	port.dev.Stop(ethdev.StopReset)
	if impl.isolated {
		impl.setIsolate(port, false)
	}
	return nil
}

type rxgFlow struct {
	face  *Face
	index int
	queue int
}

func (*rxgFlow) IsRxGroup() {}

func (rxf *rxgFlow) NumaSocket() eal.NumaSocket {
	return rxf.face.NumaSocket()
}

func (rxf *rxgFlow) Ptr() unsafe.Pointer {
	return unsafe.Pointer(&rxf.face.priv.rxf[rxf.index].base)
}
