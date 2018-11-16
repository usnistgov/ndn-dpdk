package fwdp

/*
#include "crypto.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/appinit"
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
	c   C.FwCrypto
	dev dpdk.CryptoDev
}

func newCrypto(id int) *Crypto {
	var fwc Crypto
	fwc.ResetThreadBase()
	fwc.id = id
	return &fwc
}

func (fwc *Crypto) String() string {
	return fmt.Sprintf("crypto%d", fwc.id)
}

func (fwc *Crypto) Init(cfg CryptoConfig, ndt *ndt.Ndt, fwds []*Fwd) error {
	if e := fwc.InputBase.Init(ndt, fwds); e != nil {
		return e
	} else {
		fwc.c.output = fwc.InputBase.c
	}

	numaSocket := fwc.GetNumaSocket()

	input, e := dpdk.NewRing("crypto0_queue", cfg.InputCapacity, numaSocket, false, true)
	if e != nil {
		fwc.InputBase.Close()
		return fmt.Errorf("dpdk.NewRing: %v", e)
	} else {
		fwc.c.input = (*C.struct_rte_ring)(input.GetPtr())
	}

	opPool, e := dpdk.NewCryptoOpPool("crypto0_pool", cfg.OpPoolCapacity, cfg.OpPoolCacheSize, 0, numaSocket)
	if e != nil {
		input.Close()
		fwc.InputBase.Close()
		return fmt.Errorf("dpdk.NewCryptoOpPool: %v", e)
	} else {
		fwc.c.opPool = (*C.struct_rte_mempool)(opPool.GetPtr())
	}

	fwc.dev, e = dpdk.NewOpensslCryptoDev("crypto0_dev", 1, numaSocket)
	if e != nil {
		opPool.Close()
		input.Close()
		fwc.InputBase.Close()
		return fmt.Errorf("dpdk.NewOpensslCryptoDev: %v", e)
	} else {
		fwc.c.devId = C.uint8_t(fwc.dev.GetId())
		fwc.c.qpId = 0
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
	return fwc.StopImpl(appinit.NewStopFlag(unsafe.Pointer(&fwc.c.stop)))
}

func (fwc *Crypto) Close() error {
	fwc.InputBase.Close()
	fwc.dev.Close()
	dpdk.MempoolFromPtr(unsafe.Pointer(fwc.c.opPool)).Close()
	dpdk.RingFromPtr(unsafe.Pointer(fwc.c.input)).Close()
	return nil
}
