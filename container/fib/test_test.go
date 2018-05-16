package fib_test

import (
	"os"
	"testing"

	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
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
	Ndt *ndt.Ndt
	Fib *fib.Fib
}

func NewFixture(ndtPrefixLen, fibStartDepth, nPartitions int) (fixture *Fixture) {
	ndtCfg := ndt.Config{
		PrefixLen:  ndtPrefixLen,
		IndexBits:  16,
		SampleFreq: 32,
	}
	ndt := ndt.New(ndtCfg, []dpdk.NumaSocket{dpdk.NUMA_SOCKET_ANY})
	ndt.Randomize(nPartitions)

	fibCfg := fib.Config{
		Id:         "TestFib",
		MaxEntries: 255,
		NBuckets:   64,
		StartDepth: fibStartDepth,
	}
	partitionNumaSockets := make([]dpdk.NumaSocket, nPartitions)
	for i := range partitionNumaSockets {
		partitionNumaSockets[i] = dpdk.NUMA_SOCKET_ANY
	}
	fib, e := fib.New(fibCfg, ndt, partitionNumaSockets)
	if e != nil {
		panic(e)
	}

	return &Fixture{Ndt: ndt, Fib: fib}
}

func (fixture *Fixture) Close() error {
	fixture.Fib.Close()
	return fixture.Ndt.Close()
}

// Return number of in-use entries in FIB's underlying mempool.
func (fixture *Fixture) CountMpInUse(i int) int {
	return dpdk.MempoolFromPtr(fixture.Fib.GetPtr(i)).CountInUse()
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
