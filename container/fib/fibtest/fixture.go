package fibtest

import (
	"github.com/stretchr/testify/assert"

	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/mempool"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// Fixture is a test fixture that contains a FIB.
type Fixture struct {
	Ndt         *ndt.Ndt
	Fib         *fib.Fib
	NPartitions int
}

// NewFixture initializes a FIB test fixture.
func NewFixture(ndtPrefixLen, fibStartDepth, nPartitions int) (fixture *Fixture) {
	ndtCfg := ndt.Config{
		PrefixLen:  ndtPrefixLen,
		IndexBits:  8,
		SampleFreq: 32,
	}
	ndt := ndt.New(ndtCfg, []eal.NumaSocket{{}})
	ndt.Randomize(nPartitions)

	cfg := fib.Config{
		MaxEntries: 255,
		NBuckets:   64,
		StartDepth: fibStartDepth,
	}
	partitionSockets := make([]eal.NumaSocket, nPartitions)
	fib, e := fib.New(cfg, ndt, partitionSockets)
	if e != nil {
		panic(e)
	}

	return &Fixture{Ndt: ndt, Fib: fib, NPartitions: nPartitions}
}

// Close discards the fixture.
func (fixture *Fixture) Close() error {
	fixture.Fib.Close()
	strategycode.DestroyAll()
	return fixture.Ndt.Close()
}

// CountEntries returns number of in-use entries in FIB's underlying mempool.
func (fixture *Fixture) CountEntries() (n int) {
	urcu.Barrier()
	for partition := 0; partition < fixture.NPartitions; partition++ {
		n += mempool.FromPtr(fixture.Fib.Ptr(partition)).CountInUse()
	}
	return n
}

// MakeEntry allocates and initializes a FIB entry.
func (fixture *Fixture) MakeEntry(name string, sc strategycode.StrategyCode,
	nexthops ...iface.ID) (entry *fib.Entry) {
	entry = new(fib.Entry)
	n := ndn.ParseName(name)
	entry.SetName(n)
	entry.SetNexthops(nexthops)
	if sc != nil {
		entry.SetStrategy(sc)
	}
	return entry
}

// FindInPartitions lists the partitions that contain the given name.
func (fixture *Fixture) FindInPartitions(name ndn.Name) (partitions []int) {
	rs := urcu.NewReadSide()
	defer rs.Close()
	for partition := 0; partition < fixture.NPartitions; partition++ {
		if fixture.Fib.FindInPartition(name, partition, rs) != nil {
			partitions = append(partitions, partition)
		}
	}
	return partitions
}

// CheckEntryNames checks that the FIB contains the given names.
func (fixture *Fixture) CheckEntryNames(a *assert.Assertions, expectedInput []string) bool {
	expected := make([]string, len(expectedInput))
	for i, uri := range expectedInput {
		expected[i] = ndn.ParseName(uri).String()
	}

	entryNames := fixture.Fib.ListNames()
	actual := make([]string, len(entryNames))
	for i, name := range entryNames {
		actual[i] = name.String()
	}

	return a.ElementsMatch(expected, actual)
}
