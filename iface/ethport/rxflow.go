package ethport

/*
#include "../../csrc/ethface/face.h"
#include "../../csrc/ethface/flow-pattern.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/zap"
)

// Read rte_flow_error into Go error.
func readFlowErr(e C.struct_rte_flow_error) error {
	if e._type == C.RTE_FLOW_ERROR_TYPE_NONE {
		return eal.GetErrno()
	}

	message := C.GoString(e.message)
	if causeOffset := uintptr(e.cause); causeOffset < C.sizeof_EthFlowPattern {
		return fmt.Errorf("%d %s %d", e._type, message, causeOffset)
	}
	return fmt.Errorf("%d %s %p", e._type, message, e.cause)
}

type rxFlow struct {
	isolated        bool
	flowFlags       uint32
	availQueues     []uint16
	hasDestroyError bool
}

func (rxFlow) String() string {
	return "RxFlow"
}

func (rxFlow) List(port *Port) (list []iface.RxGroup) {
	for _, face := range port.faces {
		for _, rxf := range face.rxf {
			list = append(list, rxf)
		}
	}
	return
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
	impl.flowFlags = port.devInfo.FlowFlags()

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
	nRxQueues := max(1, face.loc.EthFaceConfig().NRxQueues)
	if nRxQueues > len(impl.availQueues) {
		return fmt.Errorf("%d RX queues requested but only %d available on Port", nRxQueues, len(impl.availQueues))
	}
	if nRxQueues > iface.MaxFaceRxThreads {
		return fmt.Errorf("number of RX queues cannot exceed %d", iface.MaxFaceRxThreads)
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
	queuesC := (*C.uint16_t)(unsafe.Pointer(unsafe.SliceData(queues)))
	locC := face.loc.EthLocatorC()
	var flowErr C.struct_rte_flow_error
	face.flow = C.EthFace_SetupFlow(face.priv, queuesC, C.int(len(queues)),
		locC.ptr(), C.bool(impl.isolated), C.EthFlowFlags(impl.flowFlags), &flowErr)
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
	return nil
}

type rxgFlow struct {
	face  *Face
	index int
	queue uint16
}

var (
	_ iface.RxGroup           = (*rxgFlow)(nil)
	_ iface.RxGroupSingleFace = (*rxgFlow)(nil)
)

func (rxf *rxgFlow) NumaSocket() eal.NumaSocket {
	return rxf.face.NumaSocket()
}

func (rxf *rxgFlow) RxGroup() (ptr unsafe.Pointer, desc string) {
	return unsafe.Pointer(&rxf.face.rxfC(rxf.index).base),
		fmt.Sprintf("EthRxFlow(face=%d,port=%d,queue=%d)", rxf.face.ID(), rxf.face.port.EthDev().ID(), rxf.queue)
}

func (rxf *rxgFlow) Faces() []iface.Face {
	return []iface.Face{rxf.face}
}

func (rxgFlow) RxGroupIsSingleFace() {}
