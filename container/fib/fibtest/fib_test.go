package fibtest

import (
	"sort"
	"testing"

	"ndn-dpdk/container/strategycode"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/ndn"
)

func TestFibInsertErase(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(0, 2, 1)
	defer fixture.Close()

	var badStrategy strategycode.StrategyCode
	strategyP := strategycode.MakeEmpty("P")
	assert.Equal(0, strategyP.CountRefs())
	strategyQ := strategycode.MakeEmpty("Q")

	fib := fixture.Fib
	assert.Equal(0, fib.Len())
	assert.Equal(0, fixture.CountMpInUse(0))

	nameA := ndn.MustParseName("/A")
	assert.Nil(fib.Find(nameA))

	_, e := fib.Insert(fixture.MakeEntry("/A", badStrategy, 2851))
	assert.Error(e) // cannot insert: entry has no strategy
	assert.Equal(0, fixture.CountMpInUse(0))

	_, e = fib.Insert(fixture.MakeEntry("/A", strategyP))
	assert.Error(e) // cannot insert: entry has no nexthop
	assert.Equal(0, fixture.CountMpInUse(0))
	assert.Equal(0, strategyP.CountRefs())

	isNew, e := fib.Insert(fixture.MakeEntry("/A", strategyP, 4076))
	assert.NoError(e)
	assert.True(isNew)
	assert.Equal(1, fib.Len())
	assert.Equal(1, fixture.CountMpInUse(0))
	assert.Equal(1, strategyP.CountRefs())

	isNew, e = fib.Insert(fixture.MakeEntry("/A", strategyP, 3092))
	assert.NoError(e)
	assert.False(isNew)
	assert.Equal(1, fib.Len())
	assert.True(fixture.CountMpInUse(0) >= 1)
	assert.True(strategyP.CountRefs() >= 1)
	urcu.Barrier()
	assert.Equal(1, fixture.CountMpInUse(0))
	assert.Equal(1, strategyP.CountRefs())
	entryA := fib.Find(nameA)
	require.NotNil(entryA)
	assert.True(entryA.GetName().Equal(nameA))
	seqNum1 := entryA.GetSeqNum()

	isNew, e = fib.Insert(fixture.MakeEntry("/A", strategyQ, 3092))
	assert.NoError(e)
	assert.False(isNew)
	assert.Equal(1, fib.Len())
	assert.True(fixture.CountMpInUse(0) >= 1)
	assert.True(strategyP.CountRefs() >= 0)
	assert.Equal(1, strategyQ.CountRefs())
	urcu.Barrier()
	assert.Equal(0, strategyP.CountRefs())
	assert.Equal(1, fixture.CountMpInUse(0))
	assert.Equal(1, strategyQ.CountRefs())

	entryA = fib.Find(nameA)
	require.NotNil(entryA)
	assert.True(entryA.GetName().Equal(nameA))
	seqNum2 := entryA.GetSeqNum()
	assert.NotEqual(seqNum1, seqNum2)
	names := fib.ListNames()
	require.Len(names, 1)
	assert.True(names[0].Equal(nameA))

	assert.NoError(fib.Erase(nameA))
	assert.Equal(0, fib.Len())
	assert.Nil(fib.Find(nameA))
	assert.Len(fib.ListNames(), 0)

	assert.Error(fib.Erase(nameA))
	assert.Equal(0, fib.Len())
	urcu.Barrier()
	assert.Equal(0, strategyQ.CountRefs())
	assert.Equal(0, fixture.CountMpInUse(0))
}

func TestFibLpm(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(0, 2, 1)
	defer fixture.Close()
	fib := fixture.Fib
	strategyP := strategycode.MakeEmpty("P")

	lpm := func(name string) int {
		entry := fib.Lpm(ndn.MustParseName(name))
		if entry == nil {
			return 0
		}
		return int(entry.GetNexthops()[0])
	}

	fib.Insert(fixture.MakeEntry("/", strategyP, 5000))
	fib.Insert(fixture.MakeEntry("/A", strategyP, 5001))
	fib.Insert(fixture.MakeEntry("/A/B/C", strategyP, 5002))
	fib.Insert(fixture.MakeEntry("/M/N", strategyP, 5003))
	fib.Insert(fixture.MakeEntry("/M/N/O", strategyP, 5004))
	fib.Insert(fixture.MakeEntry("/X/Y/Z", strategyP, 5005))
	fib.Insert(fixture.MakeEntry("/X/Y", strategyP, 5006))
	fib.Insert(fixture.MakeEntry("/X", strategyP, 5007))
	assert.Equal(8, fib.Len())
	assert.Equal(1, fib.CountVirtuals()) // '/A/B' is the only virtual entry

	names := fib.ListNames()
	assert.Len(names, 8)
	nameUris := make([]string, len(names))
	for i, name := range names {
		nameUris[i] = name.String()
	}
	sort.Strings(nameUris)
	assert.Equal([]string{"/", "/A", "/A/B/C", "/M/N", "/M/N/O", "/X", "/X/Y", "/X/Y/Z"}, nameUris)

	assert.Equal(5000, lpm("/"))
	assert.Equal(5001, lpm("/A"))
	assert.Equal(5000, lpm("/AB"))
	assert.Equal(5001, lpm("/A/B"))
	assert.Equal(5002, lpm("/A/B/C"))
	assert.Equal(5002, lpm("/A/B/C/D"))
	assert.Equal(5001, lpm("/A/B/CD"))
	assert.Equal(5000, lpm("/M"))
	assert.Equal(5003, lpm("/M/N"))
	assert.Equal(5004, lpm("/M/N/O"))
	assert.Equal(5004, lpm("/M/N/O/P"))
	assert.Equal(5005, lpm("/X/Y/Z/W"))
	assert.Equal(5005, lpm("/X/Y/Z"))
	assert.Equal(5006, lpm("/X/Y"))
	assert.Equal(5007, lpm("/X"))

	assert.NoError(fib.Erase(ndn.MustParseName("/")))
	assert.Equal(7, fib.Len())
	assert.Equal(1, fib.CountVirtuals())

	assert.NoError(fib.Erase(ndn.MustParseName("/A/B/C")))
	assert.Equal(6, fib.Len())
	assert.Equal(0, fib.CountVirtuals()) // '/A/B' is gone

	assert.NoError(fib.Erase(ndn.MustParseName("/M/N")))
	assert.NoError(fib.Erase(ndn.MustParseName("/X/Y")))
	assert.Equal(4, fib.Len())
	assert.Equal(2, fib.CountVirtuals()) // '/M/N' and '/X/Y' become virtual
	assert.Len(fib.ListNames(), 4)
}
