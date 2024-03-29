package fwdptest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go4.org/must"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibtestenv"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// Fixture is a test fixture for forwarder data plane.
type Fixture struct {
	require  *require.Assertions
	StepUnit time.Duration

	DataPlane *fwdp.DataPlane
	Ndt       *ndt.Ndt
	Fib       *fib.Fib
}

// NewFixture creates a Fixture.
func NewFixture(t testing.TB, modifyConfig ...func(cfg *fwdp.Config)) (f *Fixture) {
	f = &Fixture{
		require:  require.New(t),
		StepUnit: 50 * time.Millisecond,
	}
	if ealtestenv.UsingThreads {
		f.StepUnit = 200 * time.Millisecond
	}

	var cfg fwdp.Config
	cfg.LCoreAlloc = ealthread.Config{
		fwdp.RoleInput:  {LCores: []int{eal.Workers[0].ID()}},
		fwdp.RoleOutput: {LCores: []int{eal.Workers[1].ID()}},
		fwdp.RoleCrypto: {LCores: []int{eal.Workers[2].ID()}},
		fwdp.RoleFwd:    {LCores: []int{eal.Workers[3].ID(), eal.Workers[4].ID()}},
	}

	cfg.Crypto.InputCapacity = 64
	cfg.Crypto.OpPoolCapacity = 1023

	cfg.Disk.Malloc = true

	cfg.Fib.Capacity = 65535
	cfg.Fib.NBuckets = 256
	cfg.Fib.StartDepth = 8

	cfg.Pcct.PcctCapacity = 65535
	cfg.Pcct.CsMemoryCapacity = 16384
	cfg.Pcct.CsIndirectCapacity = 16384

	cfg.LatencySampleInterval = 1

	for _, m := range modifyConfig {
		m(&cfg)
	}

	dp, e := fwdp.New(cfg)
	f.require.NoError(e)
	f.DataPlane = dp
	f.Ndt = dp.Ndt()
	f.Fib = dp.Fib()

	t.Cleanup(func() {
		must.Close(f.DataPlane)
		strategycode.DestroyAll()
	})
	return f
}

// StepDelay delays a small amount of time for packet forwarding.
func (f *Fixture) StepDelay() {
	time.Sleep(f.StepUnit)
}

// SetFibEntry inserts or replaces a FIB entry.
func (f *Fixture) SetFibEntry(name string, strategy string, nexthops ...iface.ID) {
	f.SetFibEntryParams(name, strategy, nil, nexthops...)
}

// SetFibEntryParams inserts or replaces a FIB entry with SgInit params.
func (f *Fixture) SetFibEntryParams(name string, strategy string, params map[string]any, nexthops ...iface.ID) {
	sc := strategycode.Find(strategy)
	if sc == nil {
		var e error
		sc, e = strategycode.LoadFile(strategy, "")
		f.require.NoError(e)
	}

	entry := fibtestenv.MakeEntry(name, sc, nexthops...)
	entry.Params = params
	e := f.Fib.Insert(entry)
	f.require.NoError(e)
}

// ReadFibCounters returns counters of specified FIB entry.
func (f *Fixture) ReadFibCounters(name string) (cnt fibdef.EntryCounters) {
	f.StepDelay()
	entry := f.Fib.Find(ndn.ParseName(name))
	if entry == nil {
		return
	}
	return entry.Counters()
}

// SumCounter reads a counter from all FwFwds and computes the sum.
func (f *Fixture) SumCounter(getCounter func(fwd *fwdp.Fwd) uint64) (n uint64) {
	for _, fwd := range f.DataPlane.Fwds() {
		n += getCounter(fwd)
	}
	return n
}
