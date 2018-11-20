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

// TX loop for faces that enabled thread-safe TX.
type TxLoop struct {
	dpdk.ThreadBase
	c          C.TxLoop
	numaSocket dpdk.NumaSocket
}

func NewTxLoop(faces ...IFace) (txl *TxLoop) {
	txl = new(TxLoop)
	txl.ResetThreadBase()
	dpdk.InitStopFlag(unsafe.Pointer(&txl.c.stop))
	txl.numaSocket = dpdk.NUMA_SOCKET_ANY
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
		C.TxLoop_Run(&txl.c)
		return 0
	})
}

func (txl *TxLoop) Stop() error {
	return txl.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&txl.c.stop)))
}

func (txl *TxLoop) Close() error {
	return nil
}

func (txl *TxLoop) AddFace(faces ...IFace) {
	rs := urcu.NewReadSide()
	defer rs.Close()
	for _, face := range faces {
		txl.numaSocket = face.GetNumaSocket()
		faceC := face.getPtr()
		C.cds_hlist_add_head_rcu(&faceC.threadSafeTxNode, &txl.c.head)
	}
}

func (txl *TxLoop) RemoveFace(faces ...IFace) {
	rs := urcu.NewReadSide()
	defer rs.Close()
	for _, face := range faces {
		faceC := face.getPtr()
		C.cds_hlist_del_rcu(&faceC.threadSafeTxNode)
	}
}
