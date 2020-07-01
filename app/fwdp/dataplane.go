package fwdp

/*
#include "../../csrc/fwdp/fwd.h"
#include "../../csrc/fwdp/strategy.h"
*/
import "C"
import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/container/pktqueue"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/createface"
)

type Config struct {
	Ndt      ndt.Config         // NDT config
	Fib      fib.Config         // FIB config
	Pcct     pcct.Config        // PCCT config template (Id and NumaSocket ignored)
	Suppress pit.SuppressConfig // PIT suppression config

	Crypto            CryptoConfig
	FwdInterestQueue  pktqueue.Config
	FwdDataQueue      pktqueue.Config
	FwdNackQueue      pktqueue.Config
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

	dp.la.Allocator = &eal.LCoreAlloc
	if e = dp.la.Alloc(); e != nil {
		return nil, e
	}

	{
		inputLCores := append([]eal.LCore{}, dp.la.Inputs...)
		if dp.la.Crypto.Valid() {
			inputLCores = append(inputLCores, dp.la.Crypto)
		}
		dp.ndt = ndt.New(cfg.Ndt, eal.ListNumaSocketsOfLCores(inputLCores))
		dp.ndt.Randomize(len(dp.la.Fwds))
	}

	if dp.fib, e = fib.New("FIB", cfg.Fib, dp.ndt, eal.ListNumaSocketsOfLCores(dp.la.Fwds)); e != nil {
		dp.Close()
		return nil, fmt.Errorf("fib.New: %v", e)
	}

	for i, lc := range dp.la.Fwds {
		fwd := newFwd(i)
		fwd.SetLCore(lc)
		if e := fwd.Init(dp.fib, cfg.Pcct, cfg.FwdInterestQueue, cfg.FwdDataQueue, cfg.FwdNackQueue,
			cfg.LatencySampleFreq, cfg.Suppress); e != nil {
			dp.Close()
			return nil, fmt.Errorf("Fwd.Init(%d): %v", i, e)
		}
		dp.fwds = append(dp.fwds, fwd)
	}

	for i, lc := range dp.la.Inputs {
		fwi := newInput(i, lc, dp.ndt, dp.fwds)
		dp.inputs = append(dp.inputs, fwi)
	}

	if dp.la.Crypto.Valid() {
		fwc, e := newCrypto(len(dp.inputs), dp.la.Crypto, cfg.Crypto, dp.ndt, dp.fwds)
		if e != nil {
			dp.Close()
			return nil, fmt.Errorf("Crypto.Init(): %v", e)
		}
		dp.crypto = fwc
	}

	return dp, nil
}

func (dp *DataPlane) Launch() error {
	for _, txLCore := range dp.la.Outputs {
		txl := iface.NewTxLoop(txLCore.NumaSocket())
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
