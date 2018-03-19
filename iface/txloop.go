package iface

/*
#include "txloop.h"
*/
import "C"

type ITxLooper interface {
	TxLoop()
	StopTxLoop() error
}

// TX loop for faces that enabled thread-safe TX.
type TxLooper struct {
	c        C.FaceTxLoop
	stopWait chan struct{}
}

func NewTxLooper(faces ...Face) (txl *TxLooper) {
	if len(faces) != 1 {
		panic("NewTxLooper currently requires exactly one face")
	}

	txl = new(TxLooper)
	txl.stopWait = make(chan struct{})

	txl.c.head = (*C.Face)(faces[0].GetPtr())
	return txl
}

func (txl *TxLooper) TxLoop() {
	C.FaceTxLoop_Run(&txl.c)
	txl.c.stop = false
	txl.stopWait <- struct{}{}
}

func (txl *TxLooper) StopTxLoop() error {
	txl.c.stop = true
	<-txl.stopWait
	return nil
}
