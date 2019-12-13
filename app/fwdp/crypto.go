package fwdp

/*
#include "crypto.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/container/ndt"
	"ndn-dpdk/dpdk"
)

type CryptoConfig struct {
	InputCapacity   int
	OpPoolCapacity  int
	OpPoolCacheSize int
}

type Crypto struct {
	InputBase
	dpdk.ThreadBase
	c    C.FwCrypto
	devS dpdk.CryptoDev
	devM dpdk.CryptoDev
}

func newCrypto(id int, lc dpdk.LCore) *Crypto {
	var fwc Crypto
	fwc.ResetThreadBase()
	fwc.id = id
	fwc.SetLCore(lc)
	dpdk.InitStopFlag(unsafe.Pointer(&fwc.c.stop))
	return &fwc
}

func (fwc *Crypto) String() string {
	return fmt.Sprintf("crypto%d", fwc.id)
}

func (fwc *Crypto) Init(cfg CryptoConfig, ndt *ndt.Ndt, fwds []*Fwd) error {
	numaSocket := fwc.GetNumaSocket()
	if e := fwc.InputBase.Init(ndt, fwds, numaSocket); e != nil {
		return e
	} else {
		fwc.c.output = fwc.InputBase.c
	}

	input, e := dpdk.NewRing(fwc.String()+"_queue", cfg.InputCapacity, numaSocket, false, true)
	if e != nil {
		fwc.InputBase.Close()
		return fmt.Errorf("dpdk.NewRing: %v", e)
	} else {
		fwc.c.input = (*C.struct_rte_ring)(input.GetPtr())
	}

	opPool, e := dpdk.NewCryptoOpPool(fwc.String()+"_pool", cfg.OpPoolCapacity, cfg.OpPoolCacheSize, 0, numaSocket)
	if e != nil {
		input.Close()
		fwc.InputBase.Close()
		return fmt.Errorf("dpdk.NewCryptoOpPool: %v", e)
	} else {
		fwc.c.opPool = (*C.struct_rte_mempool)(opPool.GetPtr())
	}

	fwc.devS, e = dpdk.CryptoDevDriverPref_SingleSeg.Create(fmt.Sprintf("fwc%ds", fwc.id), 1, numaSocket)
	if e != nil {
		opPool.Close()
		input.Close()
		fwc.InputBase.Close()
		return fmt.Errorf("dpdk.CryptoDevDriverPref_SingleSeg.Create: %v", e)
	} else {
		qp, _ := fwc.devS.GetQueuePair(0)
		qp.CopyToC(unsafe.Pointer(&fwc.c.singleSeg))
	}

	fwc.devM, e = dpdk.CryptoDevDriverPref_MultiSeg.Create(fmt.Sprintf("fwc%dm", fwc.id), 1, numaSocket)
	if e != nil {
		fwc.devS.Close()
		opPool.Close()
		input.Close()
		fwc.InputBase.Close()
		return fmt.Errorf("dpdk.CryptoDevDriverPref_MultiSeg.Create: %v", e)
	} else {
		qp, _ := fwc.devM.GetQueuePair(0)
		qp.CopyToC(unsafe.Pointer(&fwc.c.multiSeg))
	}

	for _, fwd := range fwds {
		fwd.c.crypto = fwc.c.input
	}

	return nil
}

func (fwc *Crypto) Launch() error {
	return fwc.LaunchImpl(func() int {
		C.FwCrypto_Run(&fwc.c)
		return 0
	})
}

func (fwc *Crypto) Stop() error {
	return fwc.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&fwc.c.stop)))
}

func (fwc *Crypto) Close() error {
	fwc.InputBase.Close()
	fwc.devM.Close()
	fwc.devS.Close()
	dpdk.MempoolFromPtr(unsafe.Pointer(fwc.c.opPool)).Close()
	dpdk.RingFromPtr(unsafe.Pointer(fwc.c.input)).Close()
	return nil
}
