package fwdp

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
)

const (
	roleInput  = "RX"
	roleOutput = "TX"
	roleCrypto = "CRYPTO"
	roleFwd    = "FWD"
)

// Config contains data plane configuration.
type Config struct {
	Ndt      ndt.Config         // NDT config
	Fib      fibdef.Config      // FIB config
	Pcct     pcct.Config        // PCCT config template
	Suppress pit.SuppressConfig // PIT suppression config

	Crypto            CryptoConfig
	FwdInterestQueue  iface.PktQueueConfig
	FwdDataQueue      iface.PktQueueConfig
	FwdNackQueue      iface.PktQueueConfig
	LatencySampleFreq int // latency sample frequency, between 0 and 30
}

// DataPlane represents the forwarder data plane.
type DataPlane struct {
	ndt    *ndt.Ndt
	fib    *fib.Fib
	inputs []*Input
	crypto *Crypto
	fwds   []*Fwd
}

// New creates and launches forwarder data plane.
func New(cfg Config) (dp *DataPlane, e error) {
	dp = new(DataPlane)

	faceSockets := append(eal.NumaSocketsOf(ethdev.List()), eal.NumaSocket{})
	lcRxTx := ealthread.DefaultAllocator.AllocGroup([]string{roleInput, roleOutput}, faceSockets)
	lcCrypto := ealthread.DefaultAllocator.Alloc(roleCrypto, eal.NumaSocket{})
	lcFwds := ealthread.DefaultAllocator.AllocMax(roleFwd)
	if len(lcRxTx) == 0 || len(lcFwds) == 0 {
		return nil, ealthread.ErrNoLCore
	}

	{
		lcInputs := append([]eal.LCore{lcCrypto}, lcRxTx[0]...)
		dp.ndt = ndt.New(cfg.Ndt, eal.NumaSocketsOf(lcInputs))
		dp.ndt.Randomize(len(lcFwds))
	}

	for _, lc := range lcRxTx[1] {
		txl := iface.NewTxLoop(lc.NumaSocket())
		txl.SetLCore(lc)
		txl.Launch()
	}

	var fibFwds []fib.LookupThread
	for i, lc := range lcFwds {
		fwd := newFwd(i)
		if e = fwd.Init(lc, cfg.Pcct, cfg.FwdInterestQueue, cfg.FwdDataQueue, cfg.FwdNackQueue,
			cfg.LatencySampleFreq, cfg.Suppress); e != nil {
			dp.Close()
			return nil, fmt.Errorf("Fwd.Init(%d): %w", i, e)
		}
		dp.fwds = append(dp.fwds, fwd)
		fibFwds = append(fibFwds, fwd)
	}

	if dp.fib, e = fib.New(cfg.Fib, fibFwds); e != nil {
		dp.Close()
		return nil, fmt.Errorf("fib.New: %w", e)
	}

	{
		dp.crypto, e = newCrypto(len(dp.inputs), lcCrypto, cfg.Crypto, dp.ndt, dp.fwds)
		if e != nil {
			dp.Close()
			return nil, fmt.Errorf("Crypto.Init(): %w", e)
		}
		dp.crypto.Launch()
	}

	for _, fwd := range dp.fwds {
		fwd.Launch()
	}

	for i, lc := range lcRxTx[0] {
		fwi := newInput(i, lc, dp.ndt, dp.fwds)
		dp.inputs = append(dp.inputs, fwi)
		fwi.rxl.Launch()
	}

	return dp, nil
}

// Close stops the data plane and releases resources.
func (dp *DataPlane) Close() error {
	iface.CloseAll()
	if dp.crypto != nil {
		dp.crypto.Close()
	}
	for _, fwd := range dp.fwds {
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
	ealthread.DefaultAllocator.Clear()
	return nil
}
