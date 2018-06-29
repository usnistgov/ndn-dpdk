package cstest

import (
	//"fmt"
	"testing"
	//"time"

	//"ndn-dpdk/dpdk"
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

	indirect := fixture.Find(ndntestutil.MakeInterest("/A/B", ndn.CanBePrefixFlag))
	require.NotNil(indirect)
	assert.False(indirect.IsDirect())

	indirect = fixture.Find(ndntestutil.MakeInterest("/A/B/C", ndn.CanBePrefixFlag))
	assert.Nil(indirect)

	ok = fixture.Insert(ndntestutil.MakeInterest("/A/B/C", ndn.CanBePrefixFlag),
		ndntestutil.MakeData("/A/B/C/D"))
	assert.True(ok)
	assert.Equal(3, fixture.Cs.Len())

	indirect = fixture.Find(ndntestutil.MakeInterest("/A/B", ndn.CanBePrefixFlag))
	require.NotNil(indirect)
	assert.False(indirect.IsDirect())

	indirect = fixture.Find(ndntestutil.MakeInterest("/A/B/C", ndn.CanBePrefixFlag))
	require.NotNil(indirect)
	assert.False(indirect.IsDirect())
	assert.Len(indirect.GetDirect().ListIndirects(), 2)

	indirect = fixture.Find(ndntestutil.MakeInterest("/A/B", ndn.MustBeFreshFlag))
	assert.Nil(indirect) // no match because CanBePrefix=0
	assert.Equal(3, fixture.Cs.Len())

	indirect = fixture.Find(ndntestutil.MakeInterest("/A/B"))
	assert.Nil(indirect)              // no match because CanBePrefix=0
	assert.Equal(2, fixture.Cs.Len()) // erasing CS entry to make room for PIT entry

	ok = fixture.Insert(
		ndntestutil.MakeInterest("/A/B", ndn.CanBePrefixFlag,
			ndn.FHDelegation{1, "/F"}, ndn.ActiveFHDelegation(0)),
		ndntestutil.MakeData("/A/B/C/D"))
	assert.True(ok)
	assert.Equal(4, fixture.Cs.Len())

	ok = fixture.Insert(
		ndntestutil.MakeInterest("/A/B/C", ndn.CanBePrefixFlag,
			ndn.FHDelegation{1, "/F"}, ndn.ActiveFHDelegation(0)),
		ndntestutil.MakeData("/A/B/C/D"))
	assert.True(ok)
	assert.Equal(5, fixture.Cs.Len())
}
