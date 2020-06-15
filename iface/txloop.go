package iface

/*
#include "../csrc/iface/txloop.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk/eal"
)

// LCoreAlloc role for TxLoop.
const LCoreRole_TxLoop = "TX"

// TX loop.
type TxLoop struct {
	eal.ThreadBase
	c          *C.TxLoop
	numaSocket eal.NumaSocket
	faces      map[FaceId]IFace
}

func NewTxLoop(numaSocket eal.NumaSocket) (txl *TxLoop) {
	txl = new(TxLoop)
	txl.c = (*C.TxLoop)(eal.Zmalloc("TxLoop", C.sizeof_TxLoop, numaSocket))
	eal.InitStopFlag(unsafe.Pointer(&txl.c.stop))
	txl.numaSocket = numaSocket
	txl.faces = make(map[FaceId]IFace)
	return txl
}

func (txl *TxLoop) GetNumaSocket() eal.NumaSocket {
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

func (txl *TxLoop) ListFaces() (list []FaceId) {
	for faceId := range txl.faces {
		list = append(list, faceId)
	}
	return list
}

func (txl *TxLoop) AddFace(face IFace) {
	rs := urcu.NewReadSide()
	defer rs.Close()

	txl.faces[face.GetFaceId()] = face
	faceC := face.getPtr()
	C.cds_hlist_add_head_rcu(&faceC.txlNode, &txl.c.head)
}

func (txl *TxLoop) RemoveFace(face IFace) {
	rs := urcu.NewReadSide()
	defer rs.Close()

	if _, ok := txl.faces[face.GetFaceId()]; !ok {
		return
	}

	delete(txl.faces, face.GetFaceId())
	faceC := face.getPtr()
	C.cds_hlist_del_rcu(&faceC.txlNode)

	urcu.Barrier()
}
