package iface

/*
#include "../csrc/iface/txloop.h"
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
		faces:  make(map[ID]Face),
	}
	txl.Thread = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.TxLoop_Run), unsafe.Pointer(txl.c)),
		ealthread.InitStopFlag(unsafe.Pointer(&txl.c.stop)),
	)
	return txl
}

type txLoop struct {
	ealthread.Thread
	c      *C.TxLoop
	socket eal.NumaSocket
	faces  map[ID]Face
}

func (txl *txLoop) ThreadRole() string {
	return "TX"
}

func (txl *txLoop) NumaSocket() eal.NumaSocket {
	return txl.socket
}

func (txl *txLoop) Close() error {
	txl.Stop()
	eal.Free(txl.c)
	return nil
}

func (txl *txLoop) CountFaces() int {
	return eal.CallMain(func() int {
		return len(txl.faces)
	}).(int)
}

func (txl *txLoop) AddFace(face Face) {
	eal.CallMain(func() {
		if txl.faces[face.ID()] != nil {
			return
		}
		txl.faces[face.ID()] = face

		faceC := face.ptr()
		C.cds_hlist_add_head_rcu(&faceC.txlNode, &txl.c.head)
	})
}

func (txl *txLoop) RemoveFace(face Face) {
	eal.CallMain(func() {
		if txl.faces[face.ID()] == nil {
			return
		}
		delete(txl.faces, face.ID())

		faceC := face.ptr()
		C.cds_hlist_del_rcu(&faceC.txlNode)
	})
	urcu.Barrier()
}
