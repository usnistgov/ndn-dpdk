package fibtest

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/container/ndt/ndtupdater"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestRelocate(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(2, 4, 4)
	defer fixture.Close()
	ndt := fixture.Ndt
	fib := fixture.Fib
	strategyP := strategycode.MakeEmpty("P")

	name0 := ndn.ParseName("/")
	nameA := ndn.ParseName("/A")
	var nameAB, nameCDW, nameEFXYZ ndn.Name
	var indexAB, indexCDW, indexEFXYZ uint64
	for indexAB == indexCDW || indexCDW == indexEFXYZ || indexEFXYZ == indexAB {
		suffix := "_" + strconv.FormatUint(rand.Uint64(), 16)
		nameAB = ndn.ParseName("/A/B" + suffix)
		nameCDW = ndn.ParseName("/C/D" + suffix + "/W")
		nameEFXYZ = ndn.ParseName("/E/F" + suffix + "/X/Y/Z")
		indexAB = ndt.IndexOfName(nameAB)
		indexCDW = ndt.IndexOfName(nameCDW)
		indexEFXYZ = ndt.IndexOfName(nameEFXYZ)
	}
	ndt.Update(indexAB, 1)
	ndt.Update(indexCDW, 2)
	ndt.Update(indexEFXYZ, 3)

	fib.Insert(fixture.MakeEntry(name0.String(), strategyP, 5000))
	assert.Equal([]int{0, 1, 2, 3}, fixture.FindInPartitions(name0))
	assert.Equal(1, fib.Len())
	assert.Equal(4, fixture.CountEntries()) // replicated /

	fib.Insert(fixture.MakeEntry(nameA.String(), strategyP, 5001))
	assert.Equal([]int{0, 1, 2, 3}, fixture.FindInPartitions(nameA))
	assert.Equal(2, fib.Len())
	assert.Equal(8, fixture.CountEntries()) // replicated /,/A

	fib.Insert(fixture.MakeEntry(nameAB.String(), strategyP, 5002))
	assert.Equal([]int{1}, fixture.FindInPartitions(nameAB))
	assert.Equal(3, fib.Len())
	assert.Equal(9, fixture.CountEntries()) // replicated /,/A

	fib.Insert(fixture.MakeEntry(nameCDW.String(), strategyP, 5003))
	assert.Equal([]int{2}, fixture.FindInPartitions(nameCDW))
	assert.Equal(4, fib.Len())
	assert.Equal(10, fixture.CountEntries()) // replicated /,/A

	fib.Insert(fixture.MakeEntry(nameEFXYZ.String(), strategyP, 5004))
	assert.Equal([]int{3}, fixture.FindInPartitions(nameEFXYZ))
	assert.Equal(5, fib.Len())
	assert.Equal(12, fixture.CountEntries()) // replicated /,/A; virtual /E/F/X/Y

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
	assert.Equal(5, fib.Len())
	assert.Equal(14, fixture.CountEntries()) // replicated /,/A; duplicated /E/F/X/Y/Z; duplicated & virtual /E/F/X/Y
	<-done
	assert.Equal([]int{1}, fixture.FindInPartitions(nameEFXYZ))

	index4, value4 := fixture.Ndt.Lookup(nameEFXYZ)
	assert.Equal(index4, indexEFXYZ)
	assert.Equal(uint8(1), value4)

	assert.Equal(5, fib.Len())
	assert.Equal(12, fixture.CountEntries())
}
