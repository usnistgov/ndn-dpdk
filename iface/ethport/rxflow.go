package ethport

/*
#include "../../csrc/ethface/face.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/zap"
)

// Read rte_flow_error into Go error.
func readFlowErr(e C.struct_rte_flow_error) error {
	if e._type == C.RTE_FLOW_ERROR_TYPE_NONE {
		return nil
	}

	message := C.GoString(e.message)
	if causeOffset := uintptr(e.cause); causeOffset < C.sizeof_EthFlowPattern {
		return fmt.Errorf("%d %s %d", e._type, message, causeOffset)
	}
	return fmt.Errorf("%d %s %p", e._type, message, e.cause)
}

type rxFlow struct {
	isolated        bool
	availQueues     []uint16
	hasDestroyError bool
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
	*impl = rxFlow{}
	if e := impl.setIsolate(port, true); e != nil {
		port.logger.Info("flow isolate mode unavailable", zap.Error(e))
	}

	maxRxQueues := int(port.devInfo.Max_rx_queues)
	if port.cfg.RxFlowQueues > maxRxQueues {
		return fmt.Errorf("%d RX queues requested but only %d allowed by driver", port.cfg.RxFlowQueues, maxRxQueues)
	}

	if e := port.startDev(port.cfg.RxFlowQueues, !impl.isolated); e != nil {
		return e
	}

	impl.availQueues = nil
	for _, q := range port.dev.RxQueues() {
		impl.availQueues = append(impl.availQueues, q.Queue)
	}
	return nil
}

func (impl *rxFlow) Start(face *Face) error {
	nRxQueues := math.MaxInt(1, face.loc.EthFaceConfig().NRxQueues)
	if nRxQueues > len(impl.availQueues) {
		return fmt.Errorf("%d RX queues requested but only %d available on Port", nRxQueues, len(impl.availQueues))
	}
	if nRxQueues > iface.MaxRxProcThreads {
		return fmt.Errorf("number of RX queues cannot exceed %d", iface.MaxRxProcThreads)
	}

	queues := impl.availQueues[:nRxQueues]
	if e := impl.setupFlow(face, queues); e != nil {
		face.port.logger.Warn("create RxFlow failure",
			zap.Uint16s("queues", queues),
			face.ID().ZapField("face"),
			zap.Error(e),
		)
		return e
	}

	impl.availQueues = impl.availQueues[nRxQueues:]
	face.port.logger.Info("create RxFlow success",
		zap.Uint16s("queues", queues),
		face.ID().ZapField("face"),
	)
	impl.startFlow(face, queues)
	return nil
}

func (impl *rxFlow) setupFlow(face *Face, queues []uint16) error {
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

func (impl *rxFlow) startFlow(face *Face, queues []uint16) {
	face.rxf = make([]*rxgFlow, len(queues))
	for i, q := range queues {
		rxf := &rxgFlow{
			face:  face,
			index: i,
			queue: q,
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

	if e := impl.destroyFlow(face); e != nil {
		face.port.logger.Warn("destroy RxFlow failure; new faces on this Port may not work",
			face.ID().ZapField("face"),
			zap.Error(e),
		)
		impl.hasDestroyError = true
	} else {
		face.port.logger.Info("destroy RxFlow success",
			face.ID().ZapField("face"),
		)
		for _, rxf := range face.rxf {
			impl.availQueues = append(impl.availQueues, rxf.queue)
		}
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
	impl.availQueues = nil

	port.dev.Stop(ethdev.StopReset)
	if impl.isolated {
		impl.setIsolate(port, false)
	}
	return nil
}

type rxgFlow struct {
	face  *Face
	index int
	queue uint16
}

func (*rxgFlow) IsRxGroup() {}

func (rxf *rxgFlow) NumaSocket() eal.NumaSocket {
	return rxf.face.NumaSocket()
}

func (rxf *rxgFlow) Ptr() unsafe.Pointer {
	return unsafe.Pointer(&rxf.face.priv.rxf[rxf.index].base)
}
