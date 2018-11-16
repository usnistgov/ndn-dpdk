package main

/*
#include "pktcopy-rx.h"
*/
import "C"
import (
	"fmt"
	// "unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type PktcopyRx struct {
	c    *C.PktcopyRx
	face iface.IFace
}

func NewPktcopyRx(face iface.IFace) *PktcopyRx {
	numaSocket := face.GetNumaSocket()
	var pcrx PktcopyRx
	pcrx.c = (*C.PktcopyRx)(dpdk.Zmalloc("PktcopyRx", C.sizeof_PktcopyRx, numaSocket))
	pcrx.c.headerMp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_HDR, numaSocket).GetPtr())
	pcrx.c.indirectMp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_IND, numaSocket).GetPtr())
	pcrx.face = face
	return &pcrx
}

func (pcrx *PktcopyRx) Close() error {
	dpdk.Free(pcrx.c)
	return nil
}

func (pcrx *PktcopyRx) SetDumpRing(ring dpdk.Ring) {
	log.WithField("from", pcrx.face).Info("enabling dump")
	pcrx.c.dumpRing = (*C.struct_rte_ring)(ring.GetPtr())
}

func (pcrx *PktcopyRx) AddTxFace(txFace iface.IFace) error {
	if pcrx.c.nTxFaces >= C.PKTCOPYRX_MAXTX {
		return fmt.Errorf("cannot link more than %d TX", C.PKTCOPYRX_MAXTX)
	}
	txFaceId := txFace.GetFaceId()
	log.WithFields(makeLogFields("from", pcrx.face, "to", txFaceId)).Info("connecting TX")
	pcrx.c.txFaces[pcrx.c.nTxFaces] = (C.FaceId)(txFaceId)
	pcrx.c.nTxFaces++
	return nil
}

func (pcrx *PktcopyRx) Run() int {
	// appinit.MakeRxLooper(pcrx.face).RxLoop(C.PKTCOPYRX_RXBURST_SIZE,
	// 	unsafe.Pointer(C.PktcopyRx_Rx), unsafe.Pointer(pcrx.c))
	return 0
}
