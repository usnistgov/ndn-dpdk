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
	c C.SingleTxLoop
}

func NewSingleTxLoop(face Face) (txl *SingleTxLoop) {
	txl = new(SingleTxLoop)
	txl.c.face = (*C.Face)(face.GetPtr())
	return txl
}

func (txl *SingleTxLoop) TxLoop() {
	C.SingleTxLoop_Run(&txl.c)
	txl.c.stop = false
}

func (txl *SingleTxLoop) StopTxLoop() error {
	txl.c.stop = true
	return nil
}

// TX loop for multiple faces that enabled thread-safe TX.
type MultiTxLoop struct {
	c C.MultiTxLoop
}

func NewMultiTxLoop(faces ...Face) (txl *MultiTxLoop) {
	txl = new(MultiTxLoop)

	for _, face := range faces {
		txl.AddFace(face)
	}
	return txl
}

func (txl *MultiTxLoop) TxLoop() {
	rs := urcu.NewReadSide()
	C.MultiTxLoop_Run(&txl.c)
	rs.Close()
	txl.c.stop = false
}

func (txl *MultiTxLoop) StopTxLoop() error {
	txl.c.stop = true
	return nil
}

func (txl *MultiTxLoop) AddFace(face Face) {
	faceC := (*C.Face)(face.GetPtr())
	C.cds_hlist_add_head_rcu(&faceC.threadSafeTxNode, &txl.c.head)
}

func (txl *MultiTxLoop) RemoveFace(face Face) {
	faceC := (*C.Face)(face.GetPtr())
	C.cds_hlist_del_rcu(&faceC.threadSafeTxNode)
}
