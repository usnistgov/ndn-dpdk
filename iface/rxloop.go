package iface

/*
#include "../csrc/iface/rxloop.h"
*/
import "C"
import (
	"io"
	"math"
	"sync/atomic"
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
	}
	rxl.Thread = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.RxLoop_Run), unsafe.Pointer(rxl.c)),
		ealthread.InitStopFlag(unsafe.Pointer(&rxl.c.stop)),
	)
	eal.CallMain(func() { rxLoopThreads[rxl] = true })
	return rxl
}

type rxLoop struct {
	ealthread.Thread
	c      *C.RxLoop
	socket eal.NumaSocket
	nRxgs  int32 // atomic
}

func (rxl *rxLoop) ThreadRole() string {
	return "RX"
}

func (rxl *rxLoop) NumaSocket() eal.NumaSocket {
	return rxl.socket
}

func (rxl *rxLoop) Close() error {
	rxl.Stop()
	eal.CallMain(func() { delete(rxLoopThreads, rxl) })
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
	return int(atomic.LoadInt32(&rxl.nRxgs))
}

func (rxl *rxLoop) AddRxGroup(rxg RxGroup) {
	rxgC := (*C.RxGroup)(rxg.Ptr())
	if rxgC.rxBurstOp == nil {
		log.Panic("RxGroup missing rxBurstOp")
	}

	eal.CallMain(func() {
		if mapRxgRxl[rxg] != nil {
			log.Panic("RxGroup is in another RxLoop")
		}
		mapRxgRxl[rxg] = rxl
		atomic.AddInt32(&rxl.nRxgs, 1)

		C.cds_hlist_add_head_rcu(&rxgC.rxlNode, &rxl.c.head)
	})
}

func (rxl *rxLoop) RemoveRxGroup(rxg RxGroup) {
	eal.CallMain(func() {
		if mapRxgRxl[rxg] != rxl {
			log.Panic("RxGroup is not in this RxLoop")
		}
		delete(mapRxgRxl, rxg)
		atomic.AddInt32(&rxl.nRxgs, -1)

		rxgC := (*C.RxGroup)(rxg.Ptr())
		C.cds_hlist_del_rcu(&rxgC.rxlNode)
	})
	urcu.Barrier()
}

var (
	// ChooseRxLoop customizes RxLoop selection in ActivateRxGroup.
	// This will be invoked on the main thread.
	// Return nil to use default algorithm.
	ChooseRxLoop = func(rxg RxGroup) RxLoop { return nil }

	rxLoopThreads = make(map[RxLoop]bool)
	mapRxgRxl     = make(map[RxGroup]RxLoop)
)

// ListRxLoops returns a list of RxLoops.
func ListRxLoops() (list []RxLoop) {
	eal.CallMain(func() {
		for rxl := range rxLoopThreads {
			list = append(list, rxl)
		}
	})
	return list
}

// ActivateRxGroup selects an available RxLoop and adds the RxGroup to it.
// Panics if no RxLoop is available.
func ActivateRxGroup(rxg RxGroup) {
	rxl := eal.CallMain(func() RxLoop {
		if rxl := ChooseRxLoop(rxg); rxl != nil {
			return rxl
		}

		if len(rxLoopThreads) == 0 {
			log.Panic("no RxLoop available")
		}

		rxgSocket := rxg.NumaSocket()
		var bestRxl RxLoop
		bestScore := math.MaxInt32
		for rxl := range rxLoopThreads {
			score := rxl.CountRxGroups()
			if !rxgSocket.Match(rxl.NumaSocket()) {
				score += 1000000
			}
			if score <= bestScore {
				bestRxl, bestScore = rxl, score
			}
		}
		return bestRxl
	}).(RxLoop)
	rxl.AddRxGroup(rxg)
}

// DeactivateRxGroup removes the RxGroup from the owning RxLoop.
func DeactivateRxGroup(rxg RxGroup) {
	rxl := eal.CallMain(func() RxLoop { return mapRxgRxl[rxg] }).(RxLoop)
	rxl.RemoveRxGroup(rxg)
}
