package iface

/*
#include "txloop.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
)

// LCoreAlloc role for TxLoop.
const LCoreRole_TxLoop = "TX"

// TX loop.
type TxLoop struct {
	dpdk.ThreadBase
	c          *C.TxLoop
	numaSocket dpdk.NumaSocket
	faces      map[FaceId]IFace
}

func NewTxLoop(faces ...IFace) (txl *TxLoop) {
	txl = new(TxLoop)
	txl.c = (*C.TxLoop)(dpdk.Zmalloc("TxLoop", C.sizeof_TxLoop, dpdk.NUMA_SOCKET_ANY))
	txl.ResetThreadBase()
	dpdk.InitStopFlag(unsafe.Pointer(&txl.c.stop))
	txl.numaSocket = dpdk.NUMA_SOCKET_ANY
	txl.faces = make(map[FaceId]IFace)
	txl.AddFace(faces...)
	return txl
}

func (txl *TxLoop) GetNumaSocket() dpdk.NumaSocket {
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
	return txl.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&txl.c.stop)))
}

func (txl *TxLoop) Close() error {
	dpdk.Free(txl.c)
	return nil
}

func (txl *TxLoop) AddFace(faces ...IFace) {
	rs := urcu.NewReadSide()
	defer rs.Close()
	for _, face := range faces {
		if numaSocket := face.GetNumaSocket(); numaSocket != dpdk.NUMA_SOCKET_ANY {
			txl.numaSocket = numaSocket
		}
		txl.faces[face.GetFaceId()] = face

		faceC := face.getPtr()
		C.cds_hlist_add_head_rcu(&faceC.txLoopNode, &txl.c.head)
	}
}

func (txl *TxLoop) RemoveFace(faces ...IFace) {
	rs := urcu.NewReadSide()
	defer rs.Close()
	for _, face := range faces {
		if _, ok := txl.faces[face.GetFaceId()]; !ok {
			panic("Face not in TxLoop")
		}
		delete(txl.faces, face.GetFaceId())

		faceC := face.getPtr()
		C.cds_hlist_del_rcu(&faceC.txLoopNode)
	}
	urcu.Barrier()
}
