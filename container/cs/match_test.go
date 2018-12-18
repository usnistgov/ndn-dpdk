package cs_test

import (
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

func TestPrefixMatch(t *testing.T) {
	assert, require := makeAR(t)

	fixture := NewFixture()
	defer fixture.Close()

	ok := fixture.Insert(ndntestutil.MakeInterest("/A/B", ndn.CanBePrefixFlag),
		ndntestutil.MakeData("/A/B/C/D"))
	assert.True(ok)
	assert.Equal(1, fixture.Cs.CountEntries(cs.CSL_MD))
	assert.Equal(1, fixture.Cs.CountEntries(cs.CSL_MI))

	direct := fixture.Find(ndntestutil.MakeInterest("/A/B/C/D"))
	require.NotNil(direct)
	assert.True(direct.IsDirect())
	assert.Len(direct.ListIndirects(), 1)

	indirect2 := fixture.Find(ndntestutil.MakeInterest("/A/B", ndn.CanBePrefixFlag))
	require.NotNil(indirect2)
	assert.False(indirect2.IsDirect())

	assert.Nil(fixture.Find(ndntestutil.MakeInterest("/A/B/C", ndn.CanBePrefixFlag)))

	ok = fixture.Insert(ndntestutil.MakeInterest("/A/B/C", ndn.CanBePrefixFlag),
		ndntestutil.MakeData("/A/B/C/D"))
	assert.True(ok)
	assert.Equal(1, fixture.Cs.CountEntries(cs.CSL_MD))
	assert.Equal(2, fixture.Cs.CountEntries(cs.CSL_MI))

	indirect2 = fixture.Find(ndntestutil.MakeInterest("/A/B", ndn.CanBePrefixFlag))
	require.NotNil(indirect2)
	assert.False(indirect2.IsDirect())

	indirect3 := fixture.Find(ndntestutil.MakeInterest("/A/B/C", ndn.CanBePrefixFlag))
	require.NotNil(indirect3)
	assert.False(indirect3.IsDirect())
	assert.Len(direct.ListIndirects(), 2)

	assert.Nil(fixture.Find(ndntestutil.MakeInterest("/A/B", ndn.MustBeFreshFlag))) // CanBePrefix=0
	assert.Equal(1, fixture.Cs.CountEntries(cs.CSL_MD))
	assert.Equal(2, fixture.Cs.CountEntries(cs.CSL_MI))

	assert.Nil(fixture.Find(ndntestutil.MakeInterest("/A/B"))) // CanBePrefix=0
	assert.Equal(1, fixture.Cs.CountEntries(cs.CSL_MD))        // erasing 'indirect2' to make room for PIT entry
	assert.Equal(1, fixture.Cs.CountEntries(cs.CSL_MI))
	assert.Len(direct.ListIndirects(), 1)

	fixture.Cs.Erase(*direct)
	assert.Equal(0, fixture.Cs.CountEntries(cs.CSL_MD))
	assert.Equal(0, fixture.Cs.CountEntries(cs.CSL_MI))

	ok = fixture.Insert(
		ndntestutil.MakeInterest("/A/B", ndn.CanBePrefixFlag,
			ndn.FHDelegation{1, "/F"}, ndn.ActiveFHDelegation(0)),
		ndntestutil.MakeData("/A/B/C/D"))
	assert.True(ok)
	assert.Equal(1, fixture.Cs.CountEntries(cs.CSL_MD))
	assert.Equal(1, fixture.Cs.CountEntries(cs.CSL_MI))

	ok = fixture.Insert(
		ndntestutil.MakeInterest("/A/B/C", ndn.CanBePrefixFlag,
			ndn.FHDelegation{1, "/F"}, ndn.ActiveFHDelegation(0)),
		ndntestutil.MakeData("/A/B/C/D"))
	assert.True(ok)
	assert.Equal(1, fixture.Cs.CountEntries(cs.CSL_MD))
	assert.Equal(2, fixture.Cs.CountEntries(cs.CSL_MI))
}

// TODO implicit digest test case
