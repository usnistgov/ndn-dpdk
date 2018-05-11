package fib_test

import (
	"os"
	"testing"

	"ndn-dpdk/container/fib"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

func TestMain(m *testing.M) {
	dpdktestenv.MakeDirectMp(255, 0, 2000)

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR

type Fixture struct {
	Fib *fib.Fib
}

func NewFixture(startDepth int) (fixture *Fixture) {
	cfg := fib.Config{
		Id:         "TestFib",
		MaxEntries: 255,
		NBuckets:   64,
		NumaSocket: dpdk.NUMA_SOCKET_ANY,
		StartDepth: startDepth,
	}

	fib, e := fib.New(cfg)
	if e != nil {
		panic(e)
	}

	fixture = new(Fixture)
	fixture.Fib = fib
	return fixture
}

func (fixture *Fixture) Close() error {
	return fixture.Fib.Close()
}

// Return number of in-use entries in FIB's underlying mempool.
func (fixture *Fixture) CountMpInUse() int {
	return dpdk.MempoolFromPtr(fixture.Fib.GetPtr()).CountInUse()
}

// Create a strategy with empty BPF program.
func (fixture *Fixture) MakeStrategy() (sc fib.StrategyCode) {
	sc, e := fixture.Fib.MakeEmptyStrategy()
	if e != nil {
		panic(e)
	}
	return sc
}

// Allocate and initialize a FIB entry.
func (fixture *Fixture) MakeEntry(name string, sc fib.StrategyCode,
	nexthops ...iface.FaceId) (entry *fib.Entry) {
	entry = new(fib.Entry)
	n := ndn.MustParseName(name)
	entry.SetName(n)
	entry.SetNexthops(nexthops)
	entry.SetStrategy(sc)
	return entry
}
