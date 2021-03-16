package iface

/*
#include "../csrc/iface/txloop.h"
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

// TxLoop is the output thread that processes outgoing packets on a set of faces.
// Functions are non-thread-safe.
type TxLoop interface {
	ealthread.ThreadWithRole
	eal.WithNumaSocket
	io.Closer

	CountFaces() int
	add(face Face)
	remove(face Face)
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
	txLoopThreads[txl] = true
	return txl
}

type txLoop struct {
	ealthread.Thread
	c      *C.TxLoop
	socket eal.NumaSocket
	nFaces int
}

func (txl *txLoop) ThreadRole() string {
	return "TX"
}

func (txl *txLoop) NumaSocket() eal.NumaSocket {
	return txl.socket
}

func (txl *txLoop) Close() error {
	txl.Stop()
	delete(txLoopThreads, txl)
	eal.Free(txl.c)
	return nil
}

func (txl *txLoop) CountFaces() int {
	return txl.nFaces
}

func (txl *txLoop) add(face Face) {
	id := face.ID()
	if mapFaceTxl[id] != nil {
		logger.Panic("Face is in another TxLoop")
	}
	mapFaceTxl[id] = txl
	txl.nFaces++

	faceC := (*C.Face)(face.Ptr())
	C.cds_hlist_add_head_rcu(&faceC.txlNode, &txl.c.head)
}

func (txl *txLoop) remove(face Face) {
	id := face.ID()
	if mapFaceTxl[id] != txl {
		logger.Panic("Face is not in this TxLoop")
	}
	delete(mapFaceTxl, id)
	txl.nFaces--

	faceC := (*C.Face)(face.Ptr())
	C.cds_hlist_del_rcu(&faceC.txlNode)
	urcu.Barrier()
}

var (
	// ChooseTxLoop customizes TxLoop selection in ActivateTxFace.
	// Return nil to use default algorithm.
	ChooseTxLoop = func(face Face) TxLoop { return nil }

	txLoopThreads = make(map[TxLoop]bool)
	mapFaceTxl    = make(map[ID]TxLoop)
)

// ListTxLoops returns a list of TxLoops.
func ListTxLoops() (list []TxLoop) {
	for txl := range txLoopThreads {
		list = append(list, txl)
	}
	return list
}

// ActivateTxFace selects an available TxLoop and adds the Face to it.
// Panics if no TxLoop is available.
func ActivateTxFace(face Face) {
	if txl := ChooseTxLoop(face); txl != nil {
		txl.add(face)
		return
	}
	if len(txLoopThreads) == 0 {
		logger.Panic("no TxLoop available")
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
	bestTxl.add(face)
}

// DeactivateTxFace removes the Face from the owning TxLoop.
func DeactivateTxFace(face Face) {
	mapFaceTxl[face.ID()].remove(face)
}
