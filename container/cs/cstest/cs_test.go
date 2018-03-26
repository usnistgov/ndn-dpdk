package cstest

import (
	"testing"

	"ndn-dpdk/ndn/ndntestutil"
)

func TestInsertErase(t *testing.T) {
	assert, require := makeAR(t)

	fixture := NewFixture(255)

	ok := fixture.Insert(ndntestutil.MakeInterest("/A/B"), ndntestutil.MakeData("/A/B"))
	assert.True(ok)
	assert.Equal(1, fixture.Cs.Len())
	assert.Len(fixture.Cs.List(), 1)
	assert.Zero(fixture.Pit.Len())
	assert.Equal(1, fixture.CountMpInUse())

	csEntry := fixture.Find(ndntestutil.MakeInterest("/A/B"))
	require.NotNil(csEntry)

	fixture.Cs.Erase(*csEntry)
	assert.Zero(fixture.Cs.Len())
	assert.Len(fixture.Cs.List(), 0)
	assert.Zero(fixture.CountMpInUse())
}
