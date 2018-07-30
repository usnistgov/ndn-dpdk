package cstest

import (
	"testing"

	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestPrefixMatch(t *testing.T) {
	assert, require := makeAR(t)

	fixture := NewFixture(255, 128)
	defer fixture.Close()

	ok := fixture.Insert(ndntestutil.MakeInterest("/A/B", ndn.CanBePrefixFlag),
		ndntestutil.MakeData("/A/B/C/D"))
	assert.True(ok)
	assert.Equal(2, fixture.Cs.Len())

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
	assert.Equal(3, fixture.Cs.Len())

	indirect2 = fixture.Find(ndntestutil.MakeInterest("/A/B", ndn.CanBePrefixFlag))
	require.NotNil(indirect2)
	assert.False(indirect2.IsDirect())

	indirect3 := fixture.Find(ndntestutil.MakeInterest("/A/B/C", ndn.CanBePrefixFlag))
	require.NotNil(indirect3)
	assert.False(indirect3.IsDirect())
	assert.Len(direct.ListIndirects(), 2)

	assert.Nil(fixture.Find(ndntestutil.MakeInterest("/A/B", ndn.MustBeFreshFlag))) // CanBePrefix=0
	assert.Equal(3, fixture.Cs.Len())

	assert.Nil(fixture.Find(ndntestutil.MakeInterest("/A/B"))) // CanBePrefix=0
	assert.Equal(2, fixture.Cs.Len())                          // erasing 'indirect2' to make room for PIT entry
	assert.Len(direct.ListIndirects(), 1)

	fixture.Cs.Erase(*direct)
	assert.Equal(0, fixture.Cs.Len())

	ok = fixture.Insert(
		ndntestutil.MakeInterest("/A/B", ndn.CanBePrefixFlag,
			ndn.FHDelegation{1, "/F"}, ndn.ActiveFHDelegation(0)),
		ndntestutil.MakeData("/A/B/C/D"))
	assert.True(ok)
	assert.Equal(2, fixture.Cs.Len())

	ok = fixture.Insert(
		ndntestutil.MakeInterest("/A/B/C", ndn.CanBePrefixFlag,
			ndn.FHDelegation{1, "/F"}, ndn.ActiveFHDelegation(0)),
		ndntestutil.MakeData("/A/B/C/D"))
	assert.True(ok)
	assert.Equal(3, fixture.Cs.Len())
}
