package fibtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestInsertErase(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(0, 2, 1)
	defer fixture.Close()

	var badStrategy strategycode.StrategyCode
	strategyP := strategycode.MakeEmpty("P")
	assert.Equal(1, strategyP.CountRefs())
	strategyQ := strategycode.MakeEmpty("Q")

	fib := fixture.Fib
	assert.Equal(0, fib.Len())
	assert.Equal(0, fixture.CountEntries())

	nameA := ndn.ParseName("/A")
	assert.Nil(fib.Find(nameA))

	_, e := fib.Insert(fixture.MakeEntry("/A", badStrategy, 2851))
	assert.Error(e) // cannot insert: entry has no strategy
	assert.Equal(0, fixture.CountEntries())

	_, e = fib.Insert(fixture.MakeEntry("/A", strategyP))
	assert.Error(e) // cannot insert: entry has no nexthop
	assert.Equal(0, fixture.CountEntries())
	assert.Equal(1, strategyP.CountRefs())

	isNew, e := fib.Insert(fixture.MakeEntry("/A", strategyP, 4076))
	assert.NoError(e)
	assert.True(isNew)
	assert.Equal(1, fib.Len())
	assert.Equal(1, fixture.CountEntries())
	assert.Equal(2, strategyP.CountRefs())

	isNew, e = fib.Insert(fixture.MakeEntry("/A", strategyP, 3092))
	assert.NoError(e)
	assert.False(isNew)
	assert.Equal(1, fib.Len())
	assert.True(fixture.CountEntries() >= 1)
	assert.True(strategyP.CountRefs() >= 2)
	urcu.Barrier()
	assert.Equal(1, fixture.CountEntries())
	assert.Equal(2, strategyP.CountRefs())
	entryA := fib.Find(nameA)
	require.NotNil(entryA)
	assert.Zero(entryA.GetName().Compare(nameA))
	seqNum1 := entryA.GetSeqNum()

	isNew, e = fib.Insert(fixture.MakeEntry("/A", strategyQ, 3092))
	assert.NoError(e)
	assert.False(isNew)
	assert.Equal(1, fib.Len())
	assert.True(fixture.CountEntries() >= 1)
	assert.True(strategyP.CountRefs() >= 1)
	assert.Equal(2, strategyQ.CountRefs())
	urcu.Barrier()
	assert.Equal(1, strategyP.CountRefs())
	assert.Equal(1, fixture.CountEntries())
	assert.Equal(2, strategyQ.CountRefs())

	entryA = fib.Find(nameA)
	require.NotNil(entryA)
	assert.Zero(entryA.GetName().Compare(nameA))
	seqNum2 := entryA.GetSeqNum()
	assert.NotEqual(seqNum1, seqNum2)
	fixture.CheckEntryNames(assert, []string{"/A"})

	assert.NoError(fib.Erase(nameA))
	assert.Equal(0, fib.Len())
	assert.Nil(fib.Find(nameA))
	fixture.CheckEntryNames(assert, []string{})

	assert.Error(fib.Erase(nameA))
	assert.Equal(0, fib.Len())
	urcu.Barrier()
	assert.Equal(1, strategyQ.CountRefs())
	assert.Equal(0, fixture.CountEntries())
}

func TestLpm(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(0, 2, 1)
	defer fixture.Close()
	fib := fixture.Fib
	strategyP := strategycode.MakeEmpty("P")

	lpm := func(name string) int {
		entry := fib.Lpm(ndn.ParseName(name))
		if entry == nil {
			return 0
		}
		return int(entry.GetNexthops()[0])
	}
	lpms := func() []int {
		return []int{
			lpm("/"),
			lpm("/A"),
			lpm("/AB"),
			lpm("/A/B"),
			lpm("/A/B/C"),
			lpm("/A/B/C/D"),
			lpm("/A/B/CD"),
			lpm("/E/F/G/H"),
			lpm("/E/F/I"),
			lpm("/J"),
			lpm("/J/K"),
			lpm("/J/K/L"),
			lpm("/J/K/M/N/O"),
			lpm("/U/V/W/X/Y/Z"),
			lpm("/U/V/W"),
			lpm("/U/V"),
			lpm("/U"),
		}
	}

	fib.Insert(fixture.MakeEntry("/", strategyP, 5000))
	fib.Insert(fixture.MakeEntry("/A", strategyP, 5100))
	fib.Insert(fixture.MakeEntry("/A/B/C", strategyP, 5101))   // insert virtual /A/B
	fib.Insert(fixture.MakeEntry("/E/F/G/H", strategyP, 5200)) // insert virtual /E/F
	fib.Insert(fixture.MakeEntry("/E/F/I", strategyP, 5201))   // don't update virtual /E/F
	fib.Insert(fixture.MakeEntry("/J/K", strategyP, 5300))
	fib.Insert(fixture.MakeEntry("/J/K/L", strategyP, 5301))   // insert virtual /J/K
	fib.Insert(fixture.MakeEntry("/J/K/M/N", strategyP, 5302)) // update virtual /J/K
	fib.Insert(fixture.MakeEntry("/U/V/W/X", strategyP, 5400)) // insert virtual /U/V
	fib.Insert(fixture.MakeEntry("/U/V/W", strategyP, 5401))   // don't update virtual /U/V
	fib.Insert(fixture.MakeEntry("/U/V", strategyP, 5402))
	fib.Insert(fixture.MakeEntry("/U", strategyP, 5403))

	assert.Equal(12, fib.Len())
	assert.Equal(16, fixture.CountEntries())
	fixture.CheckEntryNames(assert, []string{"/", "/A", "/A/B/C", "/E/F/G/H", "/E/F/I", "/J/K", "/J/K/L", "/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X"})
	assert.Equal([]int{5000, 5100, 5000, 5100, 5101, 5101, 5100, 5200, 5201, 5000, 5300, 5301, 5302, 5400, 5401, 5402, 5403}, lpms())

	assert.NoError(fib.Erase(ndn.ParseName("/")))
	assert.Equal(11, fib.Len())
	assert.Equal(15, fixture.CountEntries())
	fixture.CheckEntryNames(assert, []string{"/A", "/A/B/C", "/E/F/G/H", "/E/F/I", "/J/K", "/J/K/L", "/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X"})
	assert.Equal([]int{0, 5100, 0, 5100, 5101, 5101, 5100, 5200, 5201, 0, 5300, 5301, 5302, 5400, 5401, 5402, 5403}, lpms())

	assert.NoError(fib.Erase(ndn.ParseName("/A")))
	assert.Equal(10, fib.Len())
	assert.Equal(14, fixture.CountEntries())
	fixture.CheckEntryNames(assert, []string{"/A/B/C", "/E/F/G/H", "/E/F/I", "/J/K", "/J/K/L", "/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X"})
	assert.Equal([]int{0, 0, 0, 0, 5101, 5101, 0, 5200, 5201, 0, 5300, 5301, 5302, 5400, 5401, 5402, 5403}, lpms())

	assert.NoError(fib.Erase(ndn.ParseName("/A/B/C"))) // erase virtual /A/B
	assert.Equal(9, fib.Len())
	assert.Equal(12, fixture.CountEntries())
	fixture.CheckEntryNames(assert, []string{"/E/F/G/H", "/E/F/I", "/J/K", "/J/K/L", "/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X"})
	assert.Equal([]int{0, 0, 0, 0, 0, 0, 0, 5200, 5201, 0, 5300, 5301, 5302, 5400, 5401, 5402, 5403}, lpms())

	assert.NoError(fib.Erase(ndn.ParseName("/E/F/G/H"))) // update virtual /E/F
	assert.Equal(8, fib.Len())
	assert.Equal(11, fixture.CountEntries())
	fixture.CheckEntryNames(assert, []string{"/E/F/I", "/J/K", "/J/K/L", "/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X"})
	assert.Equal([]int{0, 0, 0, 0, 0, 0, 0, 0, 5201, 0, 5300, 5301, 5302, 5400, 5401, 5402, 5403}, lpms())

	assert.NoError(fib.Erase(ndn.ParseName("/E/F/I"))) // erase virtual /E/F
	assert.Equal(7, fib.Len())
	assert.Equal(9, fixture.CountEntries())
	fixture.CheckEntryNames(assert, []string{"/J/K", "/J/K/L", "/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X"})
	assert.Equal([]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5300, 5301, 5302, 5400, 5401, 5402, 5403}, lpms())

	assert.NoError(fib.Erase(ndn.ParseName("/J/K")))
	assert.Equal(6, fib.Len())
	assert.Equal(8, fixture.CountEntries())
	fixture.CheckEntryNames(assert, []string{"/J/K/L", "/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X"})
	assert.Equal([]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5301, 5302, 5400, 5401, 5402, 5403}, lpms())

	assert.NoError(fib.Erase(ndn.ParseName("/J/K/L"))) // don't update virtual /J/K
	assert.Equal(5, fib.Len())
	assert.Equal(7, fixture.CountEntries())
	fixture.CheckEntryNames(assert, []string{"/J/K/M/N", "/U", "/U/V", "/U/V/W", "/U/V/W/X"})
	assert.Equal([]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5302, 5400, 5401, 5402, 5403}, lpms())

	assert.NoError(fib.Erase(ndn.ParseName("/J/K/M/N"))) // erase virtual /J/K
	assert.Equal(4, fib.Len())
	assert.Equal(5, fixture.CountEntries())
	fixture.CheckEntryNames(assert, []string{"/U", "/U/V", "/U/V/W", "/U/V/W/X"})
	assert.Equal([]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5400, 5401, 5402, 5403}, lpms())

	assert.NoError(fib.Erase(ndn.ParseName("/U/V/W/X"))) // update virtual /U/V
	assert.Equal(3, fib.Len())
	assert.Equal(4, fixture.CountEntries())
	fixture.CheckEntryNames(assert, []string{"/U", "/U/V", "/U/V/W"})
	assert.Equal([]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5401, 5401, 5402, 5403}, lpms())

	assert.NoError(fib.Erase(ndn.ParseName("/U/V/W"))) // erase virtual /U/V
	assert.Equal(2, fib.Len())
	assert.Equal(2, fixture.CountEntries())
	fixture.CheckEntryNames(assert, []string{"/U", "/U/V"})
	assert.Equal([]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5402, 5402, 5402, 5403}, lpms())

	assert.NoError(fib.Erase(ndn.ParseName("/U/V")))
	assert.Equal(1, fib.Len())
	assert.Equal(1, fixture.CountEntries())
	fixture.CheckEntryNames(assert, []string{"/U"})
	assert.Equal([]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5403, 5403, 5403, 5403}, lpms())

	assert.NoError(fib.Erase(ndn.ParseName("/U")))
	assert.Equal(0, fib.Len())
	assert.Equal(0, fixture.CountEntries())
	fixture.CheckEntryNames(assert, []string{})
	assert.Equal([]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, lpms())
}
