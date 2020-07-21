package fwdptest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/strategy/strategyelf"
)

const nFwds = 2

// Fixture is a test fixture for forwarder data plane.
type Fixture struct {
	require  *require.Assertions
	StepUnit time.Duration

	DataPlane *fwdp.DataPlane
	Ndt       *ndt.Ndt
	Fib       *fib.Fib
}

// NewFixture creates a Fixture.
func NewFixture(t *testing.T) (fixture *Fixture) {
	fixture = new(Fixture)
	fixture.require = require.New(t)
	fixture.StepUnit = 50 * time.Millisecond
	if ealtestenv.UsingThreads {
		fixture.StepUnit = 200 * time.Millisecond
	}

	var dpCfg fwdp.Config

	dpCfg.Crypto.InputCapacity = 64
	dpCfg.Crypto.OpPoolCapacity = 1023

	dpCfg.Ndt.PrefixLen = 2
	dpCfg.Ndt.IndexBits = 16
	dpCfg.Ndt.SampleFreq = 8

	dpCfg.Fib.MaxEntries = 65535
	dpCfg.Fib.NBuckets = 256
	dpCfg.Fib.StartDepth = 8

	dpCfg.Pcct.MaxEntries = 65535
	dpCfg.Pcct.CsCapMd = 16384
	dpCfg.Pcct.CsCapMi = 16384

	dpCfg.LatencySampleFreq = 0

	theDp, e := fwdp.New(dpCfg)
	fixture.require.NoError(e)
	fixture.DataPlane = theDp
	fixture.Ndt = theDp.GetNdt()
	fixture.Fib = theDp.GetFib()

	e = theDp.Launch()
	fixture.require.NoError(e)

	return fixture
}

// Close destroys the fixture.
func (fixture *Fixture) Close() error {
	fixture.DataPlane.Close()
	strategycode.DestroyAll()
	return nil
}

// StepDelay delays a small amount of time for packet forwarding.
func (fixture *Fixture) StepDelay() {
	time.Sleep(fixture.StepUnit)
}

// SetFibEntry inserts or replaces a FIB entry.
func (fixture *Fixture) SetFibEntry(name string, strategy string, nexthops ...iface.ID) {
	var entry fib.Entry
	e := entry.SetName(ndn.ParseName(name))
	fixture.require.NoError(e)

	e = entry.SetNexthops(nexthops)
	fixture.require.NoError(e)

	entry.SetStrategy(fixture.makeStrategy(strategy))

	_, e = fixture.Fib.Insert(&entry)
	fixture.require.NoError(e)
}

// ReadFibCounters returns counters of specified FIB entry.
func (fixture *Fixture) ReadFibCounters(name string) fib.EntryCounters {
	return fixture.Fib.ReadEntryCounters(ndn.ParseName(name))
}

func (fixture *Fixture) makeStrategy(shortname string) strategycode.StrategyCode {
	if sc := strategycode.Find(shortname); sc != nil {
		return sc
	}

	elf, e := strategyelf.Load(shortname)
	fixture.require.NoError(e)

	sc, e := strategycode.Load(shortname, elf)
	fixture.require.NoError(e)

	return sc
}

// SumCounter reads a counter from all FwFwds and compute the sum.
func (fixture *Fixture) SumCounter(getCounter func(dp *fwdp.DataPlane, i int) uint64) (n uint64) {
	for i := 0; i < nFwds; i++ {
		n += getCounter(fixture.DataPlane, i)
	}
	return n
}
