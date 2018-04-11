package main

/*
#include "pktcopy-tx.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

const PktcopyTx_RingCapacity = 256

type PktcopyTx struct {
	c *C.PktcopyTx
}

func NewPktcopyTx(face iface.IFace) (pctx PktcopyTx, e error) {
	pctx.c = new(C.PktcopyTx)
	pctx.c.face = (C.FaceId)(face.GetFaceId())

	ring, e := dpdk.NewRing(fmt.Sprintf("PktcopyTx_%d", face.GetFaceId()), PktcopyTx_RingCapacity,
		face.GetNumaSocket(), false, true)
	if e != nil {
		return pctx, e
	}
	pctx.c.txRing = (*C.struct_rte_ring)(ring.GetPtr())

	return pctx, nil
}

func (pctx PktcopyTx) GetFace() iface.IFace {
	return iface.Get(iface.FaceId(pctx.c.face))
}

func (pctx PktcopyTx) GetRing() dpdk.Ring {
	return dpdk.RingFromPtr(unsafe.Pointer(pctx.c.txRing))
}

func (pctx PktcopyTx) Close() error {
	return pctx.GetRing().Close()
}

func (pctx PktcopyTx) Run() int {
	C.PktcopyTx_Run(pctx.c)
	return 0
}
