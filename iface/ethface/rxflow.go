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

type rxqState int

const (
	rxqStateIdle rxqState = iota
	rxqStateInUse
	rxqStateDeferred
)

const (
	rxfMaxPortQueues = 4 // up to C.RTE_MAX_QUEUES_PER_PORT
	rxfMaxFaceQueues = C.RXPROC_MAX_THREADS
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
	port     *Port
	isolated bool
	queues   []rxqState
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
	impl.isolated = enable
	return nil
}

func (impl *rxFlowImpl) Init() error {
	if impl.port.cfg.DisableRxFlow {
		return errors.New("disabled")
	}

	if e := impl.setIsolate(true); e != nil {
		impl.port.logger.WithError(e).Warn("flow isolated mode unavailable")
	}

	devInfo := impl.port.dev.DevInfo()
	nRxQueues := math.MinInt(int(devInfo.Max_rx_queues), rxfMaxPortQueues)
	if nRxQueues == 0 {
		return errors.New("unable to retrieve max_rx_queues")
	}

	if e := startDev(impl.port, nRxQueues, !impl.isolated); e != nil {
		return e
	}

	impl.queues = make([]rxqState, nRxQueues)
	return nil
}

func (impl *rxFlowImpl) Start(face *ethFace) error {
	maxQueues := math.MaxInt(1, face.loc.faceConfig().MaxRxQueues)
	var queues []int
	for rxq, state := range impl.queues {
		if len(queues) < maxQueues && state == rxqStateIdle {
			queues = append(queues, rxq)
		}
	}
	if len(queues) == 0 {
		// TODO reclaim deferred-destroy queues
		return errors.New("no available queue")
	}

	rxfs, e := newRxFlow(face, queues, impl.isolated)
	if e != nil {
		return e
	}

	impl.port.logger.WithFields(makeLogFields("rx-queues", queues, "face", face.ID())).Debug("create RxFlow")
	for _, rxf := range rxfs {
		impl.queues[rxf.queue] = rxqStateInUse
		iface.EmitRxGroupAdd(rxf)
	}
	return nil
}

func (impl *rxFlowImpl) Stop(face *ethFace) error {
	if face.flow == nil {
		return nil
	}

	for _, rxf := range face.rxf {
		iface.EmitRxGroupRemove(rxf)
	}

	nextState := rxqStateIdle
	if e := impl.destroyFlow(face.flow); e != nil {
		impl.port.logger.WithField("face", face.ID()).WithError(e).Debug("destroy RxFlow deferred")
		nextState = rxqStateDeferred
	} else {
		impl.port.logger.WithField("face", face.ID()).Debug("destroy RxFlow success")
	}

	for _, rxf := range face.rxf {
		impl.queues[rxf.queue] = nextState
	}

	face.flow = nil
	face.rxf = nil
	return nil
}

func (impl *rxFlowImpl) destroyFlow(flow *C.struct_rte_flow) error {
	var e C.struct_rte_flow_error
	if res := C.rte_flow_destroy(C.uint16_t(impl.port.dev.ID()), flow, &e); res != 0 {
		return readFlowErr(e)
	}
	return nil
}

func (impl *rxFlowImpl) Close() error {
	for _, face := range impl.port.faces {
		if face.flow != nil {
			impl.destroyFlow(face.flow)
		}
	}
	impl.queues = nil
	impl.port.dev.Stop(ethdev.StopReset)
	impl.setIsolate(false)
	return nil
}

type rxFlow struct {
	face  *ethFace
	index int
	queue int
}

func newRxFlow(face *ethFace, queues []int, isolated bool) ([]*rxFlow, error) {
	priv := face.priv

	cQueues := make([]C.uint16_t, len(queues))
	for i, queue := range queues {
		cQueues[i] = C.uint16_t(queue)
	}

	cLoc := face.loc.cLoc()
	var flowErr C.struct_rte_flow_error
	flow := C.EthFace_SetupFlow(priv, &cQueues[0], C.int(len(queues)), cLoc.ptr(), C.bool(isolated), &flowErr)
	if flow == nil {
		return nil, readFlowErr(flowErr)
	}

	face.rxf = make([]*rxFlow, len(queues))
	for i, queue := range queues {
		face.rxf[i] = &rxFlow{
			face:  face,
			index: i,
			queue: queue,
		}
	}
	return face.rxf, nil
}

func (*rxFlow) IsRxGroup() {}

func (rxf *rxFlow) NumaSocket() eal.NumaSocket {
	return rxf.face.NumaSocket()
}

func (rxf *rxFlow) Ptr() unsafe.Pointer {
	return unsafe.Pointer(&rxf.face.priv.rxf[rxf.index].base)
}
