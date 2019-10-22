package fibtest

import (
	"sort"

	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/container/strategycode"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type Fixture struct {
	Ndt         *ndt.Ndt
	Fib         *fib.Fib
	NPartitions int
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

	return &Fixture{Ndt: ndt, Fib: fib, NPartitions: nPartitions}
}

func (fixture *Fixture) Close() error {
	fixture.Fib.Close()
	strategycode.DestroyAll()
	return fixture.Ndt.Close()
}

// Count number of in-use entries in FIB's underlying mempool.
func (fixture *Fixture) CountEntries() (n int) {
	urcu.Barrier()
	for partition := 0; partition < fixture.NPartitions; partition++ {
		n += dpdk.MempoolFromPtr(fixture.Fib.GetPtr(partition)).CountInUse()
	}
	return n
}

// Allocate and initialize a FIB entry.
func (fixture *Fixture) MakeEntry(name string, sc strategycode.StrategyCode,
	nexthops ...iface.FaceId) (entry *fib.Entry) {
	entry = new(fib.Entry)
	n := ndn.MustParseName(name)
	entry.SetName(n)
	entry.SetNexthops(nexthops)
	if sc != nil {
		entry.SetStrategy(sc)
	}
	return entry
}

// Find what partitions contain the given name.
func (fixture *Fixture) FindInPartitions(name *ndn.Name) (partitions []int) {
	rs := urcu.NewReadSide()
	defer rs.Close()
	for partition := 0; partition < fixture.NPartitions; partition++ {
		if fixture.Fib.FindInPartition(name, partition, rs) != nil {
			partitions = append(partitions, partition)
		}
	}
	return partitions
}

func (fixture *Fixture) ListNames() (list []string) {
	names := fixture.Fib.ListNames()
	for _, name := range names {
		list = append(list, name.String())
	}
	sort.Strings(list)
	return list
}
