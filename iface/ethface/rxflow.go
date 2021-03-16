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

	devInfo := impl.port.dev.DevInfo()
	if !devInfo.CanAttemptRxFlow() {
		return fmt.Errorf("%s cannot use RxFlow", devInfo.DriverName())
	}

	if e := impl.setIsolate(true); e != nil {
		impl.port.logger.Warn("flow isolated mode unavailable",
			zap.Error(e),
		)
	}

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
	queues := impl.allocQueues(math.MaxInt(1, face.loc.faceConfig().MaxRxQueues))
	if len(queues) == 0 {
		// TODO reclaim deferred-destroy queues
		return errors.New("no available queue")
	}

	if e := impl.setupFlow(face, queues); e != nil {
		return e
	}

	impl.port.logger.Debug("create RxFlow",
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
	if e := impl.destroyFlow(face.flow); e != nil {
		impl.port.logger.Debug("destroy RxFlow deferred",
			face.ID().ZapField("face"),
			zap.Error(e),
		)
		nextState = rxqStateDeferred
	} else {
		impl.port.logger.Debug("destroy RxFlow success",
			face.ID().ZapField("face"),
		)
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
	if impl.isolated {
		impl.setIsolate(false)
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
