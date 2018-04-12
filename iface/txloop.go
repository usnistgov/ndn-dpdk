package iface

/*
#include "txloop.h"
*/
import "C"
import "ndn-dpdk/core/urcu"

type ITxLooper interface {
	// Run TxLoop.
	TxLoop()

	// Request to stop TxLoop.
	StopTxLoop() error
}

// TX loop for one face that enabled thread-safe TX.
type SingleTxLoop struct {
	c       C.SingleTxLoop
	stopped chan bool
}

func NewSingleTxLoop(face IFace) (txl *SingleTxLoop) {
	txl = new(SingleTxLoop)
	txl.c.face = face.getPtr()
	txl.stopped = make(chan bool)
	return txl
}

func (txl *SingleTxLoop) TxLoop() {
	C.SingleTxLoop_Run(&txl.c)
	txl.stopped <- true
}

func (txl *SingleTxLoop) StopTxLoop() error {
	txl.c.stop = true
	<-txl.stopped
	txl.c.stop = false
	return nil
}

// TX loop for multiple faces that enabled thread-safe TX.
type MultiTxLoop struct {
	c       C.MultiTxLoop
	stopped chan bool
}

func NewMultiTxLoop(faces ...IFace) (txl *MultiTxLoop) {
	txl = new(MultiTxLoop)
	for _, face := range faces {
		txl.AddFace(face)
	}
	txl.stopped = make(chan bool)
	return txl
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

func (txl *MultiTxLoop) AddFace(face IFace) {
	rs := urcu.NewReadSide()
	defer rs.Close()
	faceC := face.getPtr()
	C.cds_hlist_add_head_rcu(&faceC.threadSafeTxNode, &txl.c.head)
}

func (txl *MultiTxLoop) RemoveFace(face IFace) {
	rs := urcu.NewReadSide()
	defer rs.Close()
	faceC := face.getPtr()
	C.cds_hlist_del_rcu(&faceC.threadSafeTxNode)
}
