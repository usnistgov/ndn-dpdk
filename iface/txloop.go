package iface

/*
#include "../csrc/iface/txloop.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

// TxLoop is a thread to process outgoing packets.
type TxLoop struct {
	ealthread.Thread
	c      *C.TxLoop
	socket eal.NumaSocket
	faces  map[ID]Face
}

// NewTxLoop creates a TxLoop.
func NewTxLoop(socket eal.NumaSocket) *TxLoop {
	txl := &TxLoop{
		c:      (*C.TxLoop)(eal.Zmalloc("TxLoop", C.sizeof_TxLoop, socket)),
		socket: socket,
		faces:  make(map[ID]Face),
	}
	txl.Thread = ealthread.New(
		txl.main,
		ealthread.InitStopFlag(unsafe.Pointer(&txl.c.stop)),
	)
	return txl
}

// ThreadRole returns "TX" used in lcore allocator.
func (txl *TxLoop) ThreadRole() string {
	return "TX"
}

// NumaSocket returns NUMA socket of the data structures.
func (txl *TxLoop) NumaSocket() eal.NumaSocket {
	return txl.socket
}

func (txl *TxLoop) main() int {
	rs := urcu.NewReadSide()
	defer rs.Close()
	C.TxLoop_Run(txl.c)
	return 0
}

// Close stops the thread and deallocates data structures.
func (txl *TxLoop) Close() error {
	txl.Stop()
	eal.Free(txl.c)
	return nil
}

func (txl *TxLoop) ListFaces() (list []ID) {
	for faceID := range txl.faces {
		list = append(list, faceID)
	}
	return list
}

func (txl *TxLoop) AddFace(face Face) {
	rs := urcu.NewReadSide()
	defer rs.Close()

	txl.faces[face.ID()] = face
	faceC := face.ptr()
	C.cds_hlist_add_head_rcu(&faceC.txlNode, &txl.c.head)
}

func (txl *TxLoop) RemoveFace(face Face) {
	rs := urcu.NewReadSide()
	defer rs.Close()

	if _, ok := txl.faces[face.ID()]; !ok {
		return
	}

	delete(txl.faces, face.ID())
	faceC := face.ptr()
	C.cds_hlist_del_rcu(&faceC.txlNode)

	urcu.Barrier()
}
