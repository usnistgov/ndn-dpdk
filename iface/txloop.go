package iface

/*
#include "../csrc/iface/txloop.h"
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

// TxLoop is a thread to process outgoing packets on a set of faces.
type TxLoop interface {
	ealthread.ThreadWithRole
	eal.WithNumaSocket
	io.Closer

	CountFaces() int
	AddFace(face Face)
	RemoveFace(face Face)
}

// NewTxLoop creates a TxLoop.
func NewTxLoop(socket eal.NumaSocket) TxLoop {
	txl := &txLoop{
		c:      (*C.TxLoop)(eal.Zmalloc("TxLoop", C.sizeof_TxLoop, socket)),
		socket: socket,
	}
	txl.Thread = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.TxLoop_Run), unsafe.Pointer(txl.c)),
		ealthread.InitStopFlag(unsafe.Pointer(&txl.c.stop)),
	)
	eal.CallMain(func() { txLoopThreads[txl] = true })
	return txl
}

type txLoop struct {
	ealthread.Thread
	c      *C.TxLoop
	socket eal.NumaSocket
	nFaces int32 // atomic
}

func (txl *txLoop) ThreadRole() string {
	return "TX"
}

func (txl *txLoop) NumaSocket() eal.NumaSocket {
	return txl.socket
}

func (txl *txLoop) Close() error {
	txl.Stop()
	eal.CallMain(func() { delete(txLoopThreads, txl) })
	eal.Free(txl.c)
	return nil
}

func (txl *txLoop) CountFaces() int {
	return int(atomic.LoadInt32(&txl.nFaces))
}

func (txl *txLoop) AddFace(face Face) {
	eal.CallMain(func() {
		id := face.ID()
		if mapFaceTxl[id] != nil {
			log.Panic("Face is in another TxLoop")
		}
		mapFaceTxl[id] = txl
		atomic.AddInt32(&txl.nFaces, 1)

		faceC := face.ptr()
		C.cds_hlist_add_head_rcu(&faceC.txlNode, &txl.c.head)
	})
}

func (txl *txLoop) RemoveFace(face Face) {
	eal.CallMain(func() {
		id := face.ID()
		if mapFaceTxl[id] != txl {
			log.Panic("Face is not in this TxLoop")
		}
		delete(mapFaceTxl, id)
		atomic.AddInt32(&txl.nFaces, -1)

		faceC := face.ptr()
		C.cds_hlist_del_rcu(&faceC.txlNode)
	})
	urcu.Barrier()
}

var (
	// ChooseTxLoop customizes TxLoop selection in ActivateTxFace.
	// This will be invoked on the main thread.
	// Return nil to use default algorithm.
	ChooseTxLoop = func(face Face) TxLoop { return nil }

	txLoopThreads = make(map[TxLoop]bool)
	mapFaceTxl    = make(map[ID]TxLoop)
)

// ListTxLoops returns a list of TxLoops.
func ListTxLoops() (list []TxLoop) {
	eal.CallMain(func() {
		for txl := range txLoopThreads {
			list = append(list, txl)
		}
	})
	return list
}

// ActivateTxFace selects an available TxLoop and adds the Face to it.
// Panics if no TxLoop is available.
func ActivateTxFace(face Face) {
	txl := eal.CallMain(func() TxLoop {
		if txl := ChooseTxLoop(face); txl != nil {
			return txl
		}
		if len(txLoopThreads) == 0 {
			log.Panic("no TxLoop available")
		}

		faceSocket := face.NumaSocket()
		var bestTxl TxLoop
		bestScore := math.MaxInt32
		for txl := range txLoopThreads {
			score := txl.CountFaces()
			if !faceSocket.Match(txl.NumaSocket()) {
				score += 1000000
			}
			if score <= bestScore {
				bestTxl, bestScore = txl, score
			}
		}
		return bestTxl
	}).(TxLoop)
	txl.AddFace(face)
}

// DeactivateTxFace removes the Face from the owning TxLoop.
func DeactivateTxFace(face Face) {
	txl := eal.CallMain(func() TxLoop { return mapFaceTxl[face.ID()] }).(TxLoop)
	txl.RemoveFace(face)
}
