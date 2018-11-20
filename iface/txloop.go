package iface

/*
#include "txloop.h"
*/
import "C"
import (
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
)

type ITxLooper interface {
	GetNumaSocket() dpdk.NumaSocket

	// Run TxLoop.
	TxLoop()

	// Request to stop TxLoop.
	StopTxLoop() error
}

// TX loop for multiple faces that enabled thread-safe TX.
type MultiTxLoop struct {
	c          C.MultiTxLoop
	stopped    chan bool
	numaSocket dpdk.NumaSocket
}

func NewMultiTxLoop(faces ...IFace) (txl *MultiTxLoop) {
	txl = new(MultiTxLoop)
	txl.stopped = make(chan bool)
	txl.numaSocket = dpdk.NUMA_SOCKET_ANY
	txl.AddFace(faces...)
	return txl
}

func (txl *MultiTxLoop) GetNumaSocket() dpdk.NumaSocket {
	return txl.numaSocket
}

func (txl *MultiTxLoop) TxLoop() {
	rs := urcu.NewReadSide()
	defer rs.Close()
	C.MultiTxLoop_Run(&txl.c)
	txl.stopped <- true
}

func (txl *MultiTxLoop) StopTxLoop() error {
	txl.c.stop = true
	<-txl.stopped
	txl.c.stop = false
	return nil
}

func (txl *MultiTxLoop) AddFace(faces ...IFace) {
	rs := urcu.NewReadSide()
	defer rs.Close()
	for _, face := range faces {
		txl.numaSocket = face.GetNumaSocket()
		faceC := face.getPtr()
		C.cds_hlist_add_head_rcu(&faceC.threadSafeTxNode, &txl.c.head)
	}
}

func (txl *MultiTxLoop) RemoveFace(faces ...IFace) {
	rs := urcu.NewReadSide()
	defer rs.Close()
	for _, face := range faces {
		faceC := face.getPtr()
		C.cds_hlist_del_rcu(&faceC.threadSafeTxNode)
	}
}
