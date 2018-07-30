package fibtest

import (
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/container/strategycode"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type Fixture struct {
	Ndt *ndt.Ndt
	Fib *fib.Fib
}

func NewFixture(ndtPrefixLen, fibStartDepth, nPartitions int) (fixture *Fixture) {
	ndtCfg := ndt.Config{
		PrefixLen:  ndtPrefixLen,
		IndexBits:  8,
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
	strategycode.CloseAll()
	return fixture.Ndt.Close()
}

// Return number of in-use entries in FIB's underlying mempool.
func (fixture *Fixture) CountMpInUse(i int) int {
	return dpdk.MempoolFromPtr(fixture.Fib.GetPtr(i)).CountInUse()
}

// Allocate and initialize a FIB entry.
func (fixture *Fixture) MakeEntry(name string, sc strategycode.StrategyCode,
	nexthops ...iface.FaceId) (entry *fib.Entry) {
	entry = new(fib.Entry)
	n := ndn.MustParseName(name)
	entry.SetName(n)
	entry.SetNexthops(nexthops)
	entry.SetStrategy(sc)
	return entry
}

// Find what partitions contain the given name.
func (fixture *Fixture) FindInPartitions(name *ndn.Name) (partitions []int) {
	rs := urcu.NewReadSide()
	defer rs.Close()
	for partition := 0; partition < fixture.Fib.CountPartitions(); partition++ {
		if fixture.Fib.FindInPartition(name, partition, rs) != nil {
			partitions = append(partitions, partition)
		}
	}
	return partitions
}
