package fwdptest

import (
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibtestenv"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
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

	dpCfg.Fib.Capacity = 65535
	dpCfg.Fib.NBuckets = 256
	dpCfg.Fib.StartDepth = 8

	dpCfg.Pcct.MaxEntries = 65535
	dpCfg.Pcct.CsCapMd = 16384
	dpCfg.Pcct.CsCapMi = 16384

	dpCfg.LatencySampleFreq = 0

	dp, e := fwdp.New(dpCfg)
	fixture.require.NoError(e)
	fixture.DataPlane = dp
	fixture.Ndt = dp.Ndt()
	fixture.Fib = dp.Fib()

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
	e := fixture.Fib.Insert(fibtestenv.MakeEntry(name, fixture.makeStrategy(strategy), nexthops...))
	fixture.require.NoError(e)
}

// ReadFibCounters returns counters of specified FIB entry.
func (fixture *Fixture) ReadFibCounters(name string) (cnt fibdef.EntryCounters) {
	entry := fixture.Fib.Find(ndn.ParseName(name))
	if entry == nil {
		return
	}
	return entry.Counters()
}

func (fixture *Fixture) makeStrategy(shortname string) *strategycode.Strategy {
	if sc := strategycode.Find(shortname); sc != nil {
		return sc
	}

	_, thisFile, _, _ := runtime.Caller(1)
	elfFile := path.Join(path.Dir(thisFile), "../../../build/lib/bpf", "ndndpdk-strategy-"+shortname+".o")

	sc, e := strategycode.LoadFile(shortname, elfFile)
	fixture.require.NoError(e)

	return sc
}

// SumCounter reads a counter from all FwFwds and compute the sum.
func (fixture *Fixture) SumCounter(getCounter func(fwd *fwdp.Fwd) uint64) (n uint64) {
	for _, fwd := range fixture.DataPlane.Fwds() {
		n += getCounter(fwd)
	}
	return n
}
