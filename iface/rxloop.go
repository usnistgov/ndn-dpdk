package iface

/*
#include "../csrc/iface/rxloop.h"
*/
import "C"
import (
	"io"
	"math"
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

// RxLoop is the input thread that processes incoming packets on a set of RxGroups.
// Functions are non-thread-safe.
type RxLoop interface {
	eal.WithNumaSocket
	ealthread.ThreadWithRole
	ealthread.ThreadWithLoadStat
	io.Closer

	InterestDemux() *InputDemux
	DataDemux() *InputDemux
	NackDemux() *InputDemux

	CountRxGroups() int
	add(rxg RxGroup)
	remove(rxg RxGroup)
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
	rxLoopThreads[rxl] = true
	return rxl
}

type rxLoop struct {
	ealthread.Thread
	c      *C.RxLoop
	socket eal.NumaSocket
	nRxgs  int
}

func (rxl *rxLoop) NumaSocket() eal.NumaSocket {
	return rxl.socket
}

func (rxl *rxLoop) ThreadRole() string {
	return "RX"
}

func (rxl *rxLoop) ThreadLoadStat() ealthread.LoadStat {
	return ealthread.LoadStatFromPtr(unsafe.Pointer(&rxl.c.loadStat))
}

func (rxl *rxLoop) Close() error {
	rxl.Stop()
	delete(rxLoopThreads, rxl)
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
	return rxl.nRxgs
}

func (rxl *rxLoop) add(rxg RxGroup) {
	rxgC := (*C.RxGroup)(rxg.Ptr())
	if rxgC.rxBurstOp == nil {
		logger.Panic("RxGroup missing rxBurstOp")
	}

	if mapRxgRxl[rxg] != nil {
		logger.Panic("RxGroup is in another RxLoop")
	}
	mapRxgRxl[rxg] = rxl
	rxl.nRxgs++

	C.cds_hlist_add_head_rcu(&rxgC.rxlNode, &rxl.c.head)
}

func (rxl *rxLoop) remove(rxg RxGroup) {
	if mapRxgRxl[rxg] != rxl {
		logger.Panic("RxGroup is not in this RxLoop")
	}
	delete(mapRxgRxl, rxg)
	rxl.nRxgs--

	rxgC := (*C.RxGroup)(rxg.Ptr())
	C.cds_hlist_del_rcu(&rxgC.rxlNode)
	urcu.Barrier()
}

var (
	// ChooseRxLoop customizes RxLoop selection in ActivateRxGroup.
	// Return nil to use default algorithm.
	ChooseRxLoop = func(rxg RxGroup) RxLoop { return nil }

	rxLoopThreads = make(map[RxLoop]bool)
	mapRxgRxl     = make(map[RxGroup]RxLoop)
)

// ListRxLoops returns a list of RxLoops.
func ListRxLoops() (list []RxLoop) {
	for rxl := range rxLoopThreads {
		list = append(list, rxl)
	}
	return list
}

// ActivateRxGroup selects an available RxLoop and adds the RxGroup to it.
// Panics if no RxLoop is available.
func ActivateRxGroup(rxg RxGroup) {
	if rxl := ChooseRxLoop(rxg); rxl != nil {
		rxl.add(rxg)
		return
	}

	if len(rxLoopThreads) == 0 {
		logger.Panic("no RxLoop available")
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
	bestRxl.add(rxg)
}

// DeactivateRxGroup removes the RxGroup from the owning RxLoop.
func DeactivateRxGroup(rxg RxGroup) {
	mapRxgRxl[rxg].remove(rxg)
}
