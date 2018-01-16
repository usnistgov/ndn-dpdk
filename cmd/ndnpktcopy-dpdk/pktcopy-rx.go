package main

/*
#include "pktcopy-rx.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/iface"
)

type PktcopyRx struct {
	c *C.PktcopyRx
}

func NewPktcopyRx(face iface.Face) (pcrx PktcopyRx, e error) {
	pcrx.c = new(C.PktcopyRx)
	pcrx.c.face = (*C.Face)(face.GetPtr())

	numaSocket := face.GetNumaSocket()
	pcrx.c.mpIndirect = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_IND, numaSocket).GetPtr())

	return pcrx, nil
}

func (pcrx PktcopyRx) GetFace() iface.Face {
	return iface.FaceFromPtr(unsafe.Pointer(pcrx.c.face))
}

func (pcrx PktcopyRx) LinkTo(pctx PktcopyTx) error {
	if pcrx.c.nTxRings >= C.PKTCOPYRX_MAXTX {
		return fmt.Errorf("cannot link more than %d TX", C.PKTCOPYRX_MAXTX)
	}

	C.PktcopyRx_AddTxRing(pcrx.c, pctx.c.txRing)
	return nil
}

func (pcrx PktcopyRx) Run() int {
	C.PktcopyRx_Run(pcrx.c)
	return 0
}
