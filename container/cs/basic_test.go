package cs_test

import (
	"fmt"
	"testing"
	"time"

	"ndn-dpdk/container/cs"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestInsertErase(t *testing.T) {
	assert, require := makeAR(t)

	fixture := NewFixture()
	defer fixture.Close()

	ok := fixture.Insert(ndntestutil.MakeInterest("/A/B"),
		ndntestutil.MakeData("/A/B"))
	assert.True(ok)
	assert.Equal(1, fixture.Cs.CountEntries(cs.CSL_MD))
	assert.Zero(fixture.Pit.Len())
	assert.Equal(1, fixture.CountMpInUse())

	csEntry := fixture.Find(ndntestutil.MakeInterest("/A/B"))
	require.NotNil(csEntry)
	assert.False(csEntry.IsFresh(dpdk.TscNow()))

	ok = fixture.Insert(ndntestutil.MakeInterest("/A/B", ndn.MustBeFreshFlag),
		ndntestutil.MakeData("/A/B", 100*time.Millisecond))
	assert.True(ok)
	assert.Equal(1, fixture.Cs.CountEntries(cs.CSL_MD))

	csEntry = fixture.Find(ndntestutil.MakeInterest("/A/B"))
	require.NotNil(csEntry)
	csData := csEntry.GetData()
	assert.Equal("/A/B", csData.GetName().String())
	assert.Equal(100*time.Millisecond, csData.GetFreshnessPeriod())

	ok = fixture.Insert(
		ndntestutil.MakeInterest("/A/B", ndn.FHDelegation{1, "/F"}, ndn.ActiveFHDelegation(0)),
		ndntestutil.MakeData("/A/B", 200*time.Millisecond))
	assert.True(ok)
	assert.Equal(2, fixture.Cs.CountEntries(cs.CSL_MD))

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
	assert.Zero(fixture.Cs.CountEntries(cs.CSL_MD))
	assert.Zero(fixture.CountMpInUse())
}

func TestEvict(t *testing.T) {
	assert, _ := makeAR(t)

	fixture := NewFixture()
	defer fixture.Close()

	assert.Equal(CAP_MD, fixture.Cs.GetCapacity(cs.CSL_MD))
	assert.Equal(CAP_MI, fixture.Cs.GetCapacity(cs.CSL_MI))

	for i := 1; i <= 2000; i++ {
		name := fmt.Sprintf("/N/%d", i)
		ok := fixture.Insert(ndntestutil.MakeInterest(name, ndn.CanBePrefixFlag),
			ndntestutil.MakeData(name+"/Z"))
		assert.True(ok)
		assert.True(fixture.Cs.CountEntries(cs.CSL_MD) <= CAP_MD)
		assert.True(fixture.Cs.CountEntries(cs.CSL_MI) <= CAP_MI)
	}

	fixture.Cs.SetCapacity(cs.CSL_MI, 100)
	assert.Equal(100, fixture.Cs.GetCapacity(cs.CSL_MI))
	assert.True(fixture.Cs.CountEntries(cs.CSL_MI) <= 100)

	fixture.Cs.SetCapacity(cs.CSL_MD, 64)
	assert.Equal(64, fixture.Cs.GetCapacity(cs.CSL_MD))
	assert.True(fixture.Cs.CountEntries(cs.CSL_MD) <= 64)
	assert.True(fixture.Cs.CountEntries(cs.CSL_MI) <= 64)
}
