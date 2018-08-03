package cs_test

import (
	"fmt"
	"testing"
	"time"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestInsertErase(t *testing.T) {
	assert, require := makeAR(t)

	fixture := NewFixture(255, 128)
	defer fixture.Close()

	ok := fixture.Insert(ndntestutil.MakeInterest("/A/B"),
		ndntestutil.MakeData("/A/B"))
	assert.True(ok)
	assert.Equal(1, fixture.Cs.Len())
	assert.Len(fixture.Cs.List(), 1)
	assert.Zero(fixture.Pit.Len())
	assert.Equal(1, fixture.CountMpInUse())

	csEntry := fixture.Find(ndntestutil.MakeInterest("/A/B"))
	require.NotNil(csEntry)
	assert.False(csEntry.IsFresh(dpdk.TscNow()))

	ok = fixture.Insert(ndntestutil.MakeInterest("/A/B", ndn.MustBeFreshFlag),
		ndntestutil.MakeData("/A/B", 100*time.Millisecond))
	assert.True(ok)
	assert.Equal(1, fixture.Cs.Len())

	csEntry = fixture.Find(ndntestutil.MakeInterest("/A/B"))
	require.NotNil(csEntry)
	csData := csEntry.GetData()
	assert.Equal("/A/B", csData.GetName().String())
	assert.Equal(100*time.Millisecond, csData.GetFreshnessPeriod())

	ok = fixture.Insert(
		ndntestutil.MakeInterest("/A/B", ndn.FHDelegation{1, "/F"}, ndn.ActiveFHDelegation(0)),
		ndntestutil.MakeData("/A/B", 200*time.Millisecond))
	assert.True(ok)
	assert.Equal(2, fixture.Cs.Len())

	csEntry3 := fixture.Find(ndntestutil.MakeInterest("/A/B",
		ndn.FHDelegation{1, "/G"}, ndn.FHDelegation{2, "/F"}, ndn.ActiveFHDelegation(1)))
	require.NotNil(csEntry3)
	csData3 := csEntry3.GetData()
	assert.Equal("/A/B", csData3.GetName().String())
	assert.Equal(200*time.Millisecond, csData3.GetFreshnessPeriod())

	time.Sleep(10 * time.Millisecond)
	assert.NotNil(fixture.Find(ndntestutil.MakeInterest("/A/B", ndn.MustBeFreshFlag)))
	time.Sleep(120 * time.Millisecond)
	assert.Nil(fixture.Find(ndntestutil.MakeInterest("/A/B", ndn.MustBeFreshFlag)))
	assert.NotNil(fixture.Find(ndntestutil.MakeInterest("/A/B")))

	fixture.Cs.Erase(*csEntry)
	fixture.Cs.Erase(*csEntry3)
	assert.Zero(fixture.Cs.Len())
	assert.Len(fixture.Cs.List(), 0)
	assert.Zero(fixture.CountMpInUse())
}

func TestEvict(t *testing.T) {
	assert, _ := makeAR(t)

	capacity := 256
	fixture := NewFixture(511, capacity)
	defer fixture.Close()

	assert.Equal(capacity, fixture.Cs.GetCapacity())

	for i := 1; i <= 2000; i++ {
		name := fmt.Sprintf("/N/%d", i)
		ok := fixture.Insert(ndntestutil.MakeInterest(name), ndntestutil.MakeData(name))
		assert.True(ok)
		assert.True(fixture.Cs.Len() <= capacity)
		assert.Len(fixture.Cs.List(), fixture.Cs.Len())
	}

	capacity = 64
	fixture.Cs.SetCapacity(capacity)
	assert.Equal(capacity, fixture.Cs.GetCapacity())
	assert.True(fixture.Cs.Len() <= capacity)
	assert.Len(fixture.Cs.List(), fixture.Cs.Len())
}
