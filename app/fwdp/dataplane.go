package fwdp

/*
#include "fwd.h"
#include "input.h"
#include "strategy.h"
*/
import "C"
import (
	"fmt"

	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type Config struct {
	InputLCores []dpdk.LCore
	CryptoLCore dpdk.LCore
	FwdLCores   []dpdk.LCore

	Ndt  ndt.Config  // NDT config
	Fib  fib.Config  // FIB config (Id ignored)
	Pcct pcct.Config // PCCT config template (Id and NumaSocket ignored)

	Crypto            CryptoConfig
	FwdQueueCapacity  int // input-fwd queue capacity, must be power of 2
	LatencySampleFreq int // latency sample frequency, between 0 and 30
}

// Forwarder data plane.
type DataPlane struct {
	ndt    *ndt.Ndt
	fib    *fib.Fib
	inputs []*Input
	crypto *Crypto
	fwds   []*Fwd
}

func New(cfg Config) (dp *DataPlane, e error) {
	dp = new(DataPlane)

	{
		inputLCores := append([]dpdk.LCore{}, cfg.InputLCores...)
		if cfg.CryptoLCore != dpdk.LCORE_INVALID {
			inputLCores = append(inputLCores, cfg.CryptoLCore)
		}
		dp.ndt = ndt.New(cfg.Ndt, dpdk.ListNumaSocketsOfLCores(inputLCores))
		dp.ndt.Randomize(len(cfg.FwdLCores))
	}

	cfg.Fib.Id = "FIB"
	if dp.fib, e = fib.New(cfg.Fib, dp.ndt, dpdk.ListNumaSocketsOfLCores(cfg.FwdLCores)); e != nil {
		dp.Close()
		return nil, fmt.Errorf("fib.New: %v", e)
	}

	for i, lc := range cfg.FwdLCores {
		fwd := newFwd(i)
		fwd.SetLCore(lc)
		if e := fwd.Init(dp.fib, cfg.Pcct, cfg.FwdQueueCapacity, cfg.LatencySampleFreq); e != nil {
			dp.Close()
			return nil, fmt.Errorf("Fwd.Init(%d): %v", i, e)
		}
		dp.fwds = append(dp.fwds, fwd)
	}

	for i, lc := range cfg.InputLCores {
		fwi := newInput(i)
		fwi.SetLCore(lc)
		if e := fwi.Init(dp.ndt, dp.fwds); e != nil {
			dp.Close()
			return nil, fmt.Errorf("Input.Init(%d): %v", i, e)
		}
		dp.inputs = append(dp.inputs, fwi)
	}

	if cfg.CryptoLCore != dpdk.LCORE_INVALID {
		fwc := newCrypto(len(dp.inputs))
		fwc.SetLCore(cfg.CryptoLCore)
		if e := fwc.Init(cfg.Crypto, dp.ndt, dp.fwds); e != nil {
			dp.Close()
			return nil, fmt.Errorf("Crypto.Init(): %v", e)
		}
		dp.crypto = fwc
	}

	return dp, nil
}

func (dp *DataPlane) Launch() error {
	if dp.crypto != nil {
		dp.crypto.Launch()
	}
	for _, fwd := range dp.fwds {
		fwd.Launch()
	}
	return nil
}

func (dp *DataPlane) LaunchInput(rxl iface.IRxLooper) (fwi *Input, e error) {
	wantNumaSocket := rxl.GetNumaSocket()
	for _, fwi = range dp.inputs {
		if fwi.IsRunning() ||
			(wantNumaSocket != dpdk.NUMA_SOCKET_ANY && fwi.GetLCore().GetNumaSocket() != wantNumaSocket) {
			continue
		}
		fwi.rxl = rxl
		return fwi, fwi.Launch()
	}
	return nil, fmt.Errorf("no FwInput available on NUMA socket %d", wantNumaSocket)
}

func (dp *DataPlane) Stop() error {
	for _, fwi := range dp.inputs {
		if fwi.IsRunning() {
			fwi.Stop()
		}
	}
	if dp.crypto != nil {
		dp.crypto.Stop()
	}
	for _, fwd := range dp.fwds {
		fwd.Stop()
	}
	return nil
}

func (dp *DataPlane) Close() error {
	for _, fwi := range dp.inputs {
		fwi.Close()
	}
	for _, fwd := range dp.fwds {
		fwd.Close()
	}
	if dp.crypto != nil {
		dp.crypto.Close()
	}
	if dp.fib != nil {
		dp.fib.Close()
	}
	if dp.ndt != nil {
		dp.ndt.Close()
	}
	return nil
}
