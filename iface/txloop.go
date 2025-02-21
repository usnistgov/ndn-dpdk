package iface

/*
#include "../csrc/iface/txloop.h"
*/
import "C"
import (
	"io"
	"math"
	"sync"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/zyedidia/generic/set"
	"go.uber.org/zap"
)

// RoleTx is the thread role for TxLoop.
const RoleTx = "TX"

var defaultTxLoopFunc = map[bool]C.Face_TxLoopFunc{
	true:  C.Face_TxLoopFunc(C.TxLoop_Transfer_Linear),
	false: C.Face_TxLoopFunc(C.TxLoop_Transfer_Chained),
}

// TxLoop is the output thread that processes outgoing packets.
// Functions are non-thread-safe.
type TxLoop interface {
	eal.WithNumaSocket
	ealthread.ThreadWithRole
	ealthread.ThreadWithLoadStat
	io.Closer

	CountFaces() int
	Add(face Face)
	Remove(face Face)
}

// NewTxLoop creates a TxLoop.
func NewTxLoop(socket eal.NumaSocket) TxLoop {
	txl := &txLoop{
		c:      eal.Zmalloc[C.TxLoop]("TxLoop", C.sizeof_TxLoop, socket),
		socket: socket,
	}
	txl.ThreadWithCtrl = ealthread.NewThreadWithCtrl(
		cptr.Func0.C(C.TxLoop_Run, txl.c),
		unsafe.Pointer(&txl.c.ctrl),
	)
	txLoopThreads.Put(txl)
	logger.Info("TxLoop created",
		zap.Uintptr("txl-ptr", uintptr(unsafe.Pointer(txl.c))),
	)
	return txl
}

type txLoop struct {
	ealthread.ThreadWithCtrl
	c      *C.TxLoop
	socket eal.NumaSocket
	nFaces int
}

func (txl *txLoop) NumaSocket() eal.NumaSocket {
	return txl.socket
}

func (txl *txLoop) ThreadRole() string {
	return RoleTx
}

func (txl *txLoop) Close() error {
	txLoopLock.Lock()
	defer txLoopLock.Unlock()

	txl.Stop()
	txLoopThreads.Remove(txl)
	logger.Info("TxLoop closed",
		zap.Uintptr("txl-ptr", uintptr(unsafe.Pointer(txl.c))),
	)
	eal.Free(txl.c)
	return nil
}

func (txl *txLoop) CountFaces() int {
	return txl.nFaces
}

func (txl *txLoop) Add(face Face) {
	txLoopLock.Lock()
	defer txLoopLock.Unlock()

	id := face.ID()
	logEntry := logger.With(
		zap.Uintptr("txl-ptr", uintptr(unsafe.Pointer(txl.c))),
		txl.LCore().ZapField("txl-lc"),
		id.ZapField("face"),
	)

	if mapFaceTxl[id] != nil {
		logEntry.Panic("face is in another TxLoop")
	}
	mapFaceTxl[id] = txl
	txl.nFaces++

	logEntry.Debug("adding face to TxLoop")
	faceC := (*C.Face)(face.Ptr())
	C.cds_hlist_add_head_rcu(&faceC.txlNode, &txl.c.head)
}

func (txl *txLoop) Remove(face Face) {
	txLoopLock.Lock()
	defer txLoopLock.Unlock()

	id := face.ID()
	logEntry := logger.With(
		zap.Uintptr("txl-ptr", uintptr(unsafe.Pointer(txl.c))),
		txl.LCore().ZapField("txl-lc"),
		id.ZapField("face"),
	)

	if mapFaceTxl[id] != txl {
		logEntry.Panic("face is not in this TxLoop")
	}
	delete(mapFaceTxl, id)
	txl.nFaces--

	logEntry.Debug("removing face from TxLoop")
	faceC := (*C.Face)(face.Ptr())
	C.cds_hlist_del_rcu(&faceC.txlNode)
	urcu.Synchronize()
}

var (
	// ChooseTxLoop customizes TxLoop selection in ActivateTxFace.
	// Return nil to use default algorithm.
	ChooseTxLoop = func(face Face) TxLoop { return nil }

	txLoopThreads = set.NewMapset[TxLoop]()
	mapFaceTxl    = map[ID]TxLoop{}
	txLoopLock    sync.Mutex
)

// ListTxLoops returns a list of TxLoops.
func ListTxLoops() (list []TxLoop) {
	txLoopLock.Lock()
	defer txLoopLock.Unlock()
	return txLoopThreads.Keys()
}

// ActivateTxFace selects an TxLoop and adds the face to it.
// Returns chosen TxLoop.
//
// The default logic selects among existing TxLoops for the least loaded one, preferably on the
// same NUMA socket as the face. In case no TxLoop exists, one is created and launched
// automatically. This does not respect LCoreAlloc, and may panic if no LCore is available.
//
// This logic may be overridden via ChooseTxLoop.
func ActivateTxFace(face Face) TxLoop {
	if txl := ChooseTxLoop(face); txl != nil {
		txl.Add(face)
		return txl
	}

	socket := face.NumaSocket()
	if txLoopThreads.Size() == 0 {
		txl := NewTxLoop(socket)
		if e := ealthread.AllocLaunch(txl); e != nil {
			logger.Panic("no RxLoop available and cannot launch new RxLoop", zap.Error(e))
		}
		txl.Add(face)
		return txl
	}

	var bestTxl TxLoop
	bestScore := math.MaxInt32
	txLoopThreads.Each(func(txl TxLoop) {
		score := txl.CountFaces()
		if !socket.Match(txl.NumaSocket()) {
			score += 1000000
		}
		if score <= bestScore {
			bestTxl, bestScore = txl, score
		}
	})
	bestTxl.Add(face)
	return bestTxl
}

// DeactivateTxFace removes the Face from the owning TxLoop.
func DeactivateTxFace(face Face) {
	txLoopLock.Lock()
	txl := mapFaceTxl[face.ID()]
	txLoopLock.Unlock()
	txl.Remove(face)
}
