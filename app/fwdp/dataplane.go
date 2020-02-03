package fwdp

/*
#include "fwd.h"
#include "input.h"
#include "strategy.h"
*/
import "C"
import (
	"fmt"

	"ndn-dpdk/appinit"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
)

type Config struct {
	Ndt  ndt.Config  // NDT config
	Fib  fib.Config  // FIB config (Id ignored)
	Pcct pcct.Config // PCCT config template (Id and NumaSocket ignored)

	Crypto            CryptoConfig
	FwdInterestQueue  FwdQueueConfig
	FwdDataQueue      FwdQueueConfig
	FwdNackQueue      FwdQueueConfig
	LatencySampleFreq int // latency sample frequency, between 0 and 30
}

// Forwarder data plane.
type DataPlane struct {
	la     DpLCores
	ndt    *ndt.Ndt
	fib    *fib.Fib
	inputs []*Input
	crypto *Crypto
	fwds   []*Fwd
}

func New(cfg Config) (dp *DataPlane, e error) {
	dp = new(DataPlane)

	dp.la.Allocator = &dpdk.LCoreAlloc
	if e = dp.la.Alloc(); e != nil {
		return nil, e
	}

	{
		inputLCores := append([]dpdk.LCore{}, dp.la.Inputs...)
		if dp.la.Crypto != dpdk.LCORE_INVALID {
			inputLCores = append(inputLCores, dp.la.Crypto)
		}
		dp.ndt = ndt.New(cfg.Ndt, dpdk.ListNumaSocketsOfLCores(inputLCores))
		dp.ndt.Randomize(len(dp.la.Fwds))
	}

	cfg.Fib.Id = "FIB"
	if dp.fib, e = fib.New(cfg.Fib, dp.ndt, dpdk.ListNumaSocketsOfLCores(dp.la.Fwds)); e != nil {
		dp.Close()
		return nil, fmt.Errorf("fib.New: %v", e)
	}

	for i, lc := range dp.la.Fwds {
		fwd := newFwd(i)
		fwd.SetLCore(lc)
		if e := fwd.Init(dp.fib, cfg.Pcct, cfg.FwdInterestQueue, cfg.FwdDataQueue, cfg.FwdNackQueue, cfg.LatencySampleFreq); e != nil {
			dp.Close()
			return nil, fmt.Errorf("Fwd.Init(%d): %v", i, e)
		}
		dp.fwds = append(dp.fwds, fwd)
	}

	for i, lc := range dp.la.Inputs {
		fwi := newInput(i, lc)
		if e := fwi.Init(dp.ndt, dp.fwds); e != nil {
			dp.Close()
			return nil, fmt.Errorf("Input.Init(%d): %v", i, e)
		}
		dp.inputs = append(dp.inputs, fwi)
	}

	if dp.la.Crypto != dpdk.LCORE_INVALID {
		fwc := newCrypto(len(dp.inputs), dp.la.Crypto)
		if e := fwc.Init(cfg.Crypto, dp.ndt, dp.fwds); e != nil {
			dp.Close()
			return nil, fmt.Errorf("Crypto.Init(): %v", e)
		}
		dp.crypto = fwc
	}

	return dp, nil
}

func (dp *DataPlane) Launch() error {
	appinit.ProvideCreateFaceMempools()
	for _, txLCore := range dp.la.Outputs {
		txl := iface.NewTxLoop(txLCore.GetNumaSocket())
		txl.SetLCore(txLCore)
		txl.Launch()
		createface.AddTxLoop(txl)
	}
	if dp.crypto != nil {
		dp.crypto.Launch()
	}
	for _, fwd := range dp.fwds {
		fwd.Launch()
	}
	for _, fwi := range dp.inputs {
		fwi.rxl.Launch()
		createface.AddRxLoop(fwi.rxl)
	}
	return nil
}

func (dp *DataPlane) Close() error {
	createface.CloseAll()
	if dp.crypto != nil {
		dp.crypto.Stop()
		dp.crypto.Close()
	}
	for _, fwd := range dp.fwds {
		fwd.Stop()
		fwd.Close()
	}
	for _, fwi := range dp.inputs {
		fwi.Close()
	}
	if dp.fib != nil {
		dp.fib.Close()
	}
	if dp.ndt != nil {
		dp.ndt.Close()
	}
	dp.la.Close()
	return nil
}
