package iface

/*
#include "../csrc/iface/txloop.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// LCoreAlloc role for TxLoop.
const LCoreRole_TxLoop = "TX"

// TX loop.
type TxLoop struct {
	eal.ThreadBase
	c          *C.TxLoop
	numaSocket eal.NumaSocket
	faces      map[ID]Face
}

func NewTxLoop(numaSocket eal.NumaSocket) (txl *TxLoop) {
	txl = new(TxLoop)
	txl.c = (*C.TxLoop)(eal.Zmalloc("TxLoop", C.sizeof_TxLoop, numaSocket))
	eal.InitStopFlag(unsafe.Pointer(&txl.c.stop))
	txl.numaSocket = numaSocket
	txl.faces = make(map[ID]Face)
	return txl
}

func (txl *TxLoop) NumaSocket() eal.NumaSocket {
	return txl.numaSocket
}

func (txl *TxLoop) Launch() error {
	return txl.LaunchImpl(func() int {
		rs := urcu.NewReadSide()
		defer rs.Close()
		C.TxLoop_Run(txl.c)
		return 0
	})
}

func (txl *TxLoop) Stop() error {
	return txl.StopImpl(eal.NewStopFlag(unsafe.Pointer(&txl.c.stop)))
}

func (txl *TxLoop) Close() error {
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
	faceC := face.getPtr()
	C.cds_hlist_add_head_rcu(&faceC.txlNode, &txl.c.head)
}

func (txl *TxLoop) RemoveFace(face Face) {
	rs := urcu.NewReadSide()
	defer rs.Close()

	if _, ok := txl.faces[face.ID()]; !ok {
		return
	}

	delete(txl.faces, face.ID())
	faceC := face.getPtr()
	C.cds_hlist_del_rcu(&faceC.txlNode)

	urcu.Barrier()
}
