package fwdp

/*
#include "crypto.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type CryptoConfig struct {
	InputCapacity   int
	OpPoolCapacity  int
	OpPoolCacheSize int
	Socket          dpdk.NumaSocket
}

type Crypto struct {
	c   C.FwCrypto
	dev dpdk.CryptoDev
}

func NewCrypto(name string, cfg CryptoConfig) (fwc *Crypto, e error) {
	input, e := dpdk.NewRing(name+"_input", cfg.InputCapacity, cfg.Socket, false, true)
	if e != nil {
		return nil, e
	}

	opPool, e := dpdk.NewCryptoOpPool(name+"_pool", cfg.OpPoolCapacity, cfg.OpPoolCacheSize, 0, cfg.Socket)
	if e != nil {
		input.Close()
		return nil, e
	}

	fwc = new(Crypto)
	fwc.dev, e = dpdk.NewOpensslCryptoDev(name, 1, cfg.Socket)
	if e != nil {
		opPool.Close()
		input.Close()
		return nil, e
	}

	fwc.c.input = (*C.struct_rte_ring)(input.GetPtr())
	fwc.c.opPool = (*C.struct_rte_mempool)(opPool.GetPtr())
	fwc.c.devId = C.uint8_t(fwc.dev.GetId())
	fwc.c.qpId = 0
	return fwc, nil
}

func (fwc *Crypto) Close() error {
	fwc.dev.Close()
	dpdk.MempoolFromPtr(unsafe.Pointer(fwc.c.opPool)).Close()
	dpdk.RingFromPtr(unsafe.Pointer(fwc.c.input)).Close()
	return nil
}

// TODO don't disguise as iface.IRxLooper
func (fwc *Crypto) RxLoop(burstSize int, cb unsafe.Pointer, cbarg unsafe.Pointer) {
	fwc.c.output = (*C.FwInput)(cbarg)
	C.FwCrypto_Run(&fwc.c)
}

func (fwc *Crypto) StopRxLoop() error {
	fwc.c.stop = C.bool(true)
	return nil
}

func (fwc *Crypto) ListFacesInRxLoop() []iface.FaceId {
	return []iface.FaceId{1}
}
