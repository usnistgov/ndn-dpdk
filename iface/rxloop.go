package iface

/*
#include "../csrc/iface/rxloop.h"
*/
import "C"
import (
	"io"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

// RxGroup is a receive channel for a group of faces.
type RxGroup interface {
	eal.WithNumaSocket

	// IsRxGroup identifies an implementation as RxGroup.
	IsRxGroup()

	// Ptr returns *C.RxGroup pointer.
	Ptr() unsafe.Pointer
}

// RxLoop is a thread to process incoming packets on a set of RxGroups.
type RxLoop interface {
	ealthread.ThreadWithRole
	eal.WithNumaSocket
	io.Closer

	InterestDemux() *InputDemux
	DataDemux() *InputDemux
	NackDemux() *InputDemux

	CountRxGroups() int
	AddRxGroup(rxg RxGroup)
	RemoveRxGroup(rxg RxGroup)
}

// NewRxLoop creates an RxLoop.
func NewRxLoop(socket eal.NumaSocket) RxLoop {
	rxl := &rxLoop{
		c:      (*C.RxLoop)(eal.Zmalloc("RxLoop", C.sizeof_RxLoop, socket)),
		socket: socket,
		rxgs:   make(map[*C.RxGroup]RxGroup),
	}
	rxl.Thread = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.RxLoop_Run), unsafe.Pointer(rxl.c)),
		ealthread.InitStopFlag(unsafe.Pointer(&rxl.c.stop)),
	)
	return rxl
}

type rxLoop struct {
	ealthread.Thread
	c      *C.RxLoop
	socket eal.NumaSocket
	rxgs   map[*C.RxGroup]RxGroup
}

func (rxl *rxLoop) ThreadRole() string {
	return "RX"
}

func (rxl *rxLoop) NumaSocket() eal.NumaSocket {
	return rxl.socket
}

func (rxl *rxLoop) Close() error {
	rxl.Stop()
	eal.Free(rxl.c)
	return nil
}

func (rxl *rxLoop) InterestDemux() *InputDemux {
	return InputDemuxFromPtr(unsafe.Pointer(&rxl.c.demuxI))
}

func (rxl *rxLoop) DataDemux() *InputDemux {
	return InputDemuxFromPtr(unsafe.Pointer(&rxl.c.demuxD))
}

func (rxl *rxLoop) NackDemux() *InputDemux {
	return InputDemuxFromPtr(unsafe.Pointer(&rxl.c.demuxN))
}

func (rxl *rxLoop) CountRxGroups() int {
	return eal.CallMain(func() int {
		return len(rxl.rxgs)
	}).(int)
}

func (rxl *rxLoop) AddRxGroup(rxg RxGroup) {
	rxgC := (*C.RxGroup)(rxg.Ptr())
	if rxgC.rxBurstOp == nil {
		log.Panic("RxGroup missing rxBurstOp")
	}

	eal.CallMain(func() {
		if rxl.rxgs[rxgC] != nil {
			return
		}
		rxl.rxgs[rxgC] = rxg

		C.cds_hlist_add_head_rcu(&rxgC.rxlNode, &rxl.c.head)
	})
}

func (rxl *rxLoop) RemoveRxGroup(rxg RxGroup) {
	eal.CallMain(func() {
		rxgC := (*C.RxGroup)(rxg.Ptr())
		if rxl.rxgs[rxgC] == nil {
			return
		}
		delete(rxl.rxgs, rxgC)

		C.cds_hlist_del_rcu(&rxgC.rxlNode)
	})
	urcu.Barrier()
}
