package fwdp

/*
#include "crypto.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/app/inputdemux"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/dpdk"
)

type CryptoConfig struct {
	InputCapacity  int
	OpPoolCapacity int
}

type Crypto struct {
	dpdk.ThreadBase
	id     int
	c      *C.FwCrypto
	demuxD inputdemux.Demux
	devS   dpdk.CryptoDev
	devM   dpdk.CryptoDev
}

func newCrypto(id int, lc dpdk.LCore, cfg CryptoConfig, ndt *ndt.Ndt, fwds []*Fwd) (fwc *Crypto, e error) {
	socket := lc.GetNumaSocket()
	fwc = new(Crypto)
	fwc.SetLCore(lc)
	fwc.id = id
	fwc.c = (*C.FwCrypto)(dpdk.ZmallocAligned("FwCrypto", C.sizeof_FwCrypto, 1, socket))
	dpdk.InitStopFlag(unsafe.Pointer(&fwc.c.stop))

	input, e := dpdk.NewRing(fwc.String()+"_input", cfg.InputCapacity, socket, false, true)
	if e != nil {
		return nil, fmt.Errorf("dpdk.NewRing: %v", e)
	} else {
		fwc.c.input = (*C.struct_rte_ring)(input.GetPtr())
	}

	opPool, e := dpdk.NewCryptoOpPool(fwc.String()+"_pool", cfg.OpPoolCapacity, 0, socket)
	if e != nil {
		return nil, fmt.Errorf("dpdk.NewCryptoOpPool: %v", e)
	} else {
		fwc.c.opPool = (*C.struct_rte_mempool)(opPool.GetPtr())
	}

	fwc.devS, e = dpdk.CryptoDrvSingleSeg.Create(fmt.Sprintf("fwc%ds", fwc.id), 1, socket)
	if e != nil {
		return nil, fmt.Errorf("dpdk.CryptoDrvSingleSeg.Create: %v", e)
	} else {
		qp, _ := fwc.devS.GetQueuePair(0)
		qp.CopyToC(unsafe.Pointer(&fwc.c.singleSeg))
	}

	fwc.devM, e = dpdk.CryptoDrvMultiSeg.Create(fmt.Sprintf("fwc%dm", fwc.id), 1, socket)
	if e != nil {
		return nil, fmt.Errorf("dpdk.CryptoDrvMultiSeg.Create: %v", e)
	} else {
		qp, _ := fwc.devM.GetQueuePair(0)
		qp.CopyToC(unsafe.Pointer(&fwc.c.multiSeg))
	}

	fwc.demuxD = inputdemux.DemuxFromPtr(unsafe.Pointer(&fwc.c.output))
	fwc.demuxD.InitNdt(ndt, id)
	for i, fwd := range fwds {
		fwc.demuxD.SetDest(i, fwd.dataQueue)
		fwd.c.crypto = fwc.c.input
	}

	return fwc, nil
}

func (fwc *Crypto) String() string {
	return fmt.Sprintf("crypto%d", fwc.id)
}

func (fwc *Crypto) Launch() error {
	return fwc.LaunchImpl(func() int {
		C.FwCrypto_Run(fwc.c)
		return 0
	})
}

func (fwc *Crypto) Stop() error {
	return fwc.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&fwc.c.stop)))
}

func (fwc *Crypto) Close() error {
	fwc.devM.Close()
	fwc.devS.Close()
	dpdk.MempoolFromPtr(unsafe.Pointer(fwc.c.opPool)).Close()
	dpdk.RingFromPtr(unsafe.Pointer(fwc.c.input)).Close()
	dpdk.Free(unsafe.Pointer(fwc.c))
	return nil
}
