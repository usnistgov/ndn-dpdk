// Package fwdp implements the forwarder's data plane.
package fwdp

import (
	"fmt"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go4.org/must"
)

const (
	roleInput  = "RX"
	roleOutput = "TX"
	roleCrypto = "CRYPTO"
	roleFwd    = "FWD"
)

// Config contains data plane configuration.
type Config struct {
	Ndt      ndt.Config         `json:"ndt,omitempty"`
	Fib      fibdef.Config      `json:"fib,omitempty"`
	Pcct     pcct.Config        `json:"pcct,omitempty"`
	Suppress pit.SuppressConfig `json:"suppress,omitempty"`

	Crypto            CryptoConfig         `json:"crypto,omitempty"`
	FwdInterestQueue  iface.PktQueueConfig `json:"fwdInterestQueue,omitempty"`
	FwdDataQueue      iface.PktQueueConfig `json:"fwdDataQueue,omitempty"`
	FwdNackQueue      iface.PktQueueConfig `json:"fwdNackQueue,omitempty"`
	LatencySampleFreq *int                 `json:"latencySampleFreq,omitempty"` // latency sample frequency, between 0 and 30
}

func (cfg *Config) applyDefaults() {
	if cfg.FwdDataQueue.DequeueBurstSize <= 0 {
		cfg.FwdDataQueue.DequeueBurstSize = iface.MaxBurstSize
	}
	if cfg.FwdNackQueue.DequeueBurstSize <= 0 {
		cfg.FwdNackQueue.DequeueBurstSize = cfg.FwdDataQueue.DequeueBurstSize
	}
	if cfg.FwdInterestQueue.DequeueBurstSize <= 0 {
		cfg.FwdInterestQueue.DequeueBurstSize = math.MaxInt(cfg.FwdDataQueue.DequeueBurstSize/2, 1)
	}
}

func (cfg Config) latencySampleFreq() int {
	if cfg.LatencySampleFreq == nil {
		return 16
	}
	return math.MinInt(math.MaxInt(0, *cfg.LatencySampleFreq), 30)
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
	cfg.applyDefaults()
	dp = new(DataPlane)

	faceSockets := append(eal.NumaSocketsOf(ethdev.List()), eal.NumaSocket{})
	lcRxTx := ealthread.DefaultAllocator.AllocGroup([]string{roleInput, roleOutput}, faceSockets)
	lcCrypto := ealthread.DefaultAllocator.Alloc(roleCrypto, eal.NumaSocket{})
	lcFwds := ealthread.DefaultAllocator.AllocMax(roleFwd)
	if lcRxTx == nil || len(lcFwds) == 0 {
		return nil, ealthread.ErrNoLCore
	}
	lcRx, lcTx := lcRxTx[0], lcRxTx[1]

	{
		var lcInputs []eal.LCore
		lcInputs = append(lcInputs, lcRx...)
		lcInputs = append(lcInputs, lcCrypto)
		dp.ndt = ndt.New(cfg.Ndt, eal.NumaSocketsOf(lcInputs))
		dp.ndt.Randomize(len(lcFwds))
	}

	for _, lc := range lcTx {
		txl := iface.NewTxLoop(lc.NumaSocket())
		txl.SetLCore(lc)
		txl.Launch()
	}

	var fibFwds []fib.LookupThread
	for i, lc := range lcFwds {
		fwd := newFwd(i)
		if e = fwd.Init(lc, cfg.Pcct, cfg.FwdInterestQueue, cfg.FwdDataQueue, cfg.FwdNackQueue,
			cfg.latencySampleFreq(), cfg.Suppress); e != nil {
			must.Close(dp)
			return nil, fmt.Errorf("Fwd.Init(%d): %w", i, e)
		}
		dp.fwds = append(dp.fwds, fwd)
		fibFwds = append(fibFwds, fwd)
	}

	if dp.fib, e = fib.New(cfg.Fib, fibFwds); e != nil {
		must.Close(dp)
		return nil, fmt.Errorf("fib.New: %w", e)
	}

	{
		dp.crypto, e = newCrypto(len(dp.inputs), lcCrypto, cfg.Crypto, dp.ndt, dp.fwds)
		if e != nil {
			must.Close(dp)
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

// Ndt returns the NDT.
func (dp *DataPlane) Ndt() *ndt.Ndt {
	return dp.ndt
}

// Fib returns the FIB.
func (dp *DataPlane) Fib() *fib.Fib {
	return dp.fib
}

// Fwds returns a list of forwarding threads.
func (dp *DataPlane) Fwds() []*Fwd {
	return dp.fwds
}

// Close stops the data plane and releases resources.
func (dp *DataPlane) Close() error {
	iface.CloseAll()
	if dp.crypto != nil {
		must.Close(dp.crypto)
	}
	for _, fwd := range dp.fwds {
		must.Close(fwd)
	}
	for _, fwi := range dp.inputs {
		must.Close(fwi)
	}
	if dp.fib != nil {
		must.Close(dp.fib)
	}
	if dp.ndt != nil {
		must.Close(dp.ndt)
	}
	ealthread.DefaultAllocator.Clear()
	return nil
}
