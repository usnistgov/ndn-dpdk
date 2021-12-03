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

type rxFlowImpl struct {
	isolated bool
	queues   []rxqState
}

func (rxFlowImpl) Kind() RxImplKind {
	return RxImplFlow
}

// Enter or leave flow isolation mode.
func (impl *rxFlowImpl) setIsolate(port *Port, enable bool) error {
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

func (impl *rxFlowImpl) Init(port *Port) error {
	if port.devInfo.IsVDev() {
		return errors.New("cannot use RxFlow on virtual device")
	}

	if e := impl.setIsolate(port, true); e != nil {
		port.logger.Info("flow isolated mode unavailable", zap.Error(e))
	}

	nRxQueues := math.MinInt(int(port.devInfo.Max_rx_queues), port.cfg.RxFlowQueues)
	if nRxQueues == 0 {
		return errors.New("unable to retrieve max_rx_queues")
	}

	if e := port.startDev(nRxQueues, !impl.isolated); e != nil {
		return e
	}

	impl.queues = make([]rxqState, nRxQueues)
	return nil
}

func (impl *rxFlowImpl) Start(face *ethFace) error {
	queues := impl.allocQueues(math.MinInt(math.MaxInt(1, face.loc.faceConfig().MaxRxQueues), iface.MaxRxProcThreads))
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

func (impl *rxFlowImpl) allocQueues(max int) (queues []int) {
	for rxq, state := range impl.queues {
		if len(queues) < max && state == rxqStateIdle {
			queues = append(queues, rxq)
		}
	}
	return
}

func (impl *rxFlowImpl) setupFlow(face *ethFace, queues []int) error {
	var cQueues []C.uint16_t
	for _, q := range queues {
		cQueues = append(cQueues, C.uint16_t(q))
	}

	cLoc := face.loc.cLoc()
	var flowErr C.struct_rte_flow_error
	face.flow = C.EthFace_SetupFlow(face.priv, &cQueues[0], C.int(len(queues)), cLoc.ptr(), C.bool(impl.isolated), &flowErr)
	if face.flow == nil {
		return readFlowErr(flowErr)
	}
	return nil
}

func (impl *rxFlowImpl) startFlow(face *ethFace, queues []int) {
	face.rxf = make([]*rxFlow, len(queues))
	for i, queue := range queues {
		impl.queues[queue] = rxqStateInUse
		rxf := &rxFlow{
			face:  face,
			index: i,
			queue: queue,
		}
		face.rxf[i] = rxf
		iface.ActivateRxGroup(rxf)
	}
}

func (impl *rxFlowImpl) Stop(face *ethFace) error {
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

func (impl *rxFlowImpl) destroyFlow(face *ethFace) error {
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

func (impl *rxFlowImpl) Close(port *Port) error {
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

type rxFlow struct {
	face  *ethFace
	index int
	queue int
}

func (*rxFlow) IsRxGroup() {}

func (rxf *rxFlow) NumaSocket() eal.NumaSocket {
	return rxf.face.NumaSocket()
}

func (rxf *rxFlow) Ptr() unsafe.Pointer {
	return unsafe.Pointer(&rxf.face.priv.rxf[rxf.index].base)
}
