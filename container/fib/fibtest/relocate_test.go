package fibtest

import (
	"testing"
	"time"

	"ndn-dpdk/container/ndt/ndtupdater"
	"ndn-dpdk/container/strategycode"
	"ndn-dpdk/ndn"
)

func TestFibRelocate(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(2, 4, 4)
	defer fixture.Close()
	ndt := fixture.Ndt
	fib := fixture.Fib
	strategyP := strategycode.MakeEmpty()

	name0 := ndn.MustParseName("/")
	nameA := ndn.MustParseName("/A")
	nameAB := ndn.MustParseName("/A/B")
	nameCDW := ndn.MustParseName("/C/D/W")
	nameEFXYZ := ndn.MustParseName("/E/F/X/Y/Z")

	indexAB := ndt.GetIndex(ndt.ComputeHash(nameAB))
	indexCDW := ndt.GetIndex(ndt.ComputeHash(nameCDW))
	indexEFXYZ := ndt.GetIndex(ndt.ComputeHash(nameEFXYZ))
	require.NotEqual(indexAB, indexCDW)
	require.NotEqual(indexCDW, indexEFXYZ)
	require.NotEqual(indexEFXYZ, indexAB)
	ndt.Update(indexAB, 1)
	ndt.Update(indexCDW, 2)
	ndt.Update(indexEFXYZ, 3)

	fib.Insert(fixture.MakeEntry(name0.String(), strategyP, 5000))
	assert.Equal([]int{0, 1, 2, 3}, fixture.FindInPartitions(name0))
	assert.Equal(1, fib.CountEntries(false))
	assert.Equal(4, fib.CountEntries(true))
	assert.Equal(0, fib.CountVirtuals())

	fib.Insert(fixture.MakeEntry(nameA.String(), strategyP, 5001))
	assert.Equal([]int{0, 1, 2, 3}, fixture.FindInPartitions(nameA))
	assert.Equal(2, fib.CountEntries(false))
	assert.Equal(8, fib.CountEntries(true))
	assert.Equal(0, fib.CountVirtuals())

	fib.Insert(fixture.MakeEntry(nameAB.String(), strategyP, 5002))
	assert.Equal([]int{1}, fixture.FindInPartitions(nameAB))
	assert.Equal(3, fib.CountEntries(false))
	assert.Equal(9, fib.CountEntries(true))
	assert.Equal(0, fib.CountVirtuals())

	fib.Insert(fixture.MakeEntry(nameCDW.String(), strategyP, 5003))
	assert.Equal([]int{2}, fixture.FindInPartitions(nameCDW))
	assert.Equal(4, fib.CountEntries(false))
	assert.Equal(10, fib.CountEntries(true))
	assert.Equal(0, fib.CountVirtuals())

	fib.Insert(fixture.MakeEntry(nameEFXYZ.String(), strategyP, 5004))
	assert.Equal([]int{3}, fixture.FindInPartitions(nameEFXYZ))
	assert.Equal(5, fib.CountEntries(false))
	assert.Equal(11, fib.CountEntries(true))
	assert.Equal(1, fib.CountVirtuals())

	nu := ndtupdater.NdtUpdater{
		Ndt:      fixture.Ndt,
		Fib:      fixture.Fib,
		SleepFor: 100 * time.Millisecond,
	}
	done := make(chan bool)
	go func() {
		nRelocated, e := nu.Update(indexEFXYZ, 1)
		assert.NoError(e)
		assert.Equal(2, nRelocated)
		done <- true
	}()
	time.Sleep(20 * time.Millisecond)
	assert.Equal([]int{1, 3}, fixture.FindInPartitions(nameEFXYZ))
	assert.Equal(5, fib.CountEntries(false))
	assert.Equal(11, fib.CountEntries(true))
	assert.Equal(1, fib.CountVirtuals())
	<-done
	assert.Equal([]int{1}, fixture.FindInPartitions(nameEFXYZ))

	index4, value4 := fixture.Ndt.Lookup(nameEFXYZ)
	assert.Equal(index4, indexEFXYZ)
	assert.Equal(uint8(1), value4)

	assert.Equal(5, fib.CountEntries(false))
	assert.Equal(11, fib.CountEntries(true))
	assert.Equal(1, fib.CountVirtuals())
}
