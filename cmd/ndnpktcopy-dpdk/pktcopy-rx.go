package main

/*
#include "pktcopy-rx.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

const PktcopyRx_BurstSize = 64

type PktcopyRx struct {
	c    *C.PktcopyRx
	face iface.IFace
}

func NewPktcopyRx(face iface.IFace) (pcrx PktcopyRx, e error) {
	pcrx.c = new(C.PktcopyRx)
	pcrx.face = face

	numaSocket := face.GetNumaSocket()
	pcrx.c.indirectMp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_IND, numaSocket).GetPtr())

	return pcrx, nil
}

func (pcrx PktcopyRx) GetFace() iface.IFace {
	return pcrx.face
}

func (pcrx PktcopyRx) LinkTo(txRing dpdk.Ring) error {
	if pcrx.c.nTxRings >= C.PKTCOPYRX_MAXTX {
		return fmt.Errorf("cannot link more than %d TX", C.PKTCOPYRX_MAXTX)
	}

	C.PktcopyRx_AddTxRing(pcrx.c, (*C.struct_rte_ring)(txRing.GetPtr()))
	return nil
}

func (pcrx PktcopyRx) Run() int {
	appinit.MakeRxLooper(pcrx.face).RxLoop(PktcopyRx_BurstSize,
		unsafe.Pointer(C.PktcopyRx_Rx), unsafe.Pointer(pcrx.c))
	return 0
}
