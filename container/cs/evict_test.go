package cs_test

import (
	"testing"

	"ndn-dpdk/container/cs"
	"ndn-dpdk/ndn"
)

func TestEvict(t *testing.T) {
	assert, _ := makeAR(t)

	fixture := NewFixture()
	defer fixture.Close()

	assert.Equal(CAP_MD, fixture.Cs.GetCapacity(cs.CSL_MD))
	assert.Equal(CAP_MI, fixture.Cs.GetCapacity(cs.CSL_MI))

	nInserted := fixture.InsertBulk(1, 2000, "/N/%d/Z", "/N/%d", ndn.CanBePrefixFlag)
	assert.Equal(2000, nInserted)
	assert.True(fixture.Cs.CountEntries(cs.CSL_MD) <= CAP_MD)
	assert.True(fixture.Cs.CountEntries(cs.CSL_MI) <= CAP_MI)

	fixture.Cs.SetCapacity(cs.CSL_MI, 100)
	assert.Equal(100, fixture.Cs.GetCapacity(cs.CSL_MI))
	assert.True(fixture.Cs.CountEntries(cs.CSL_MI) <= 100)

	fixture.Cs.SetCapacity(cs.CSL_MD, 64)
	assert.Equal(64, fixture.Cs.GetCapacity(cs.CSL_MD))
	assert.True(fixture.Cs.CountEntries(cs.CSL_MD) <= 64)
	assert.True(fixture.Cs.CountEntries(cs.CSL_MI) <= 64)
}

func TestIndirectLru(t *testing.T) {
	assert, _ := makeAR(t)

	fixture := NewFixture()
	defer fixture.Close()

	fixture.Cs.SetCapacity(cs.CSL_MD, 500)
	fixture.Cs.SetCapacity(cs.CSL_MI, 500)

	nInserted := fixture.InsertBulk(1, 2000, "/N/%d/Z", "/N/%d", ndn.CanBePrefixFlag)
	assert.Equal(2000, nInserted)

	nFound0 := fixture.FindBulk(1701, 1800, "/N/%d", ndn.CanBePrefixFlag)
	assert.Equal(100, nFound0)

	fixture.Cs.SetCapacity(cs.CSL_MI, 200)
	nFound1 := fixture.FindBulk(1701, 1800, "/N/%d", ndn.CanBePrefixFlag)
	assert.Equal(100, nFound1)
	nFound2 := fixture.FindBulk(1901, 2000, "/N/%d", ndn.CanBePrefixFlag)
	assert.True(nFound2 < 100)
}
