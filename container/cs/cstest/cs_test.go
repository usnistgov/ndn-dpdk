package cstest

import (
	"fmt"
	"testing"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestInsertErase(t *testing.T) {
	assert, require := makeAR(t)

	fixture := NewFixture(255, 128)
	defer fixture.Close()

	// Interest MustBeFresh=0
	ok := fixture.Insert(ndntestutil.MakeInterest("/A/B"),
		ndntestutil.MakeData("/A/B"))
	assert.True(ok)
	assert.Equal(1, fixture.Cs.Len())
	assert.Len(fixture.Cs.List(), 1)
	assert.Zero(fixture.Pit.Len())
	assert.Equal(1, fixture.CountMpInUse())

	csEntry := fixture.Find(ndntestutil.MakeInterest("/A/B"))
	assert.NotNil(csEntry)
	assert.False(csEntry.IsFresh(dpdk.TscNow()))

	// Interest MustBeFresh=1
	ok = fixture.Insert(ndntestutil.MakeInterest("/A/B", ndn.MustBeFreshFlag),
		ndntestutil.MakeData("/A/B"))
	assert.True(ok)
	assert.Equal(1, fixture.Cs.Len())

	csEntry = fixture.Find(ndntestutil.MakeInterest("/A/B"))
	require.NotNil(csEntry)
	assert.Equal("/A/B", csEntry.GetData().GetName().String())

	fixture.Cs.Erase(*csEntry)
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
