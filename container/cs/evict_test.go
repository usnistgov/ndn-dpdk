package cs_test

import (
	"testing"

	"ndn-dpdk/container/cs"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/ndn"
)

func TestEvict(t *testing.T) {
	assert, _ := makeAR(t)
	var cfg pcct.Config
	cfg.CsCapMd = 400
	cfg.CsCapMi = 100
	fixture := NewFixture(cfg)
	defer fixture.Close()

	assert.Equal(400, fixture.Cs.GetCapacity(cs.CSL_MD))
	assert.Equal(100, fixture.Cs.GetCapacity(cs.CSL_MI))

	assert.Equal(2000, fixture.InsertBulk(1, 2000, "/N/%d/Z", "/N/%d", ndn.CanBePrefixFlag))
	assert.True(fixture.Cs.CountEntries(cs.CSL_MD) <= 400)
	assert.True(fixture.Cs.CountEntries(cs.CSL_MI) <= 100)

	assert.Zero(fixture.FindBulk(1, 1600, "/N/%d/Z"))
	assert.True(fixture.FindBulk(1601, 2000, "/N/%d/Z") > 300)

	assert.Equal(300, fixture.InsertBulk(2001, 2300, "/N/%d", "/N/%d"))
	assert.Zero(fixture.FindBulk(1601, 1900, "/N/%d"))
	assert.True(fixture.FindBulk(1901, 2300, "/N/%d", ndn.CanBePrefixFlag) > 300)
}

func TestIndirectLru(t *testing.T) {
	assert, _ := makeAR(t)
	var cfg pcct.Config
	cfg.CsCapMd = 600
	cfg.CsCapMi = 400
	fixture := NewFixture(cfg)
	defer fixture.Close()

	// insert 1-2000, should keep (most of) 1601-2000 indirect entries
	assert.Equal(2000, fixture.InsertBulk(1, 2000, "/N/%d/Z", "/N/%d", ndn.CanBePrefixFlag))
	// 1701-1900 become most-recently-used
	assert.Equal(200, fixture.FindBulk(1701, 1900, "/N/%d", ndn.CanBePrefixFlag))

	// insert 2001-2200, should evict 1901-2000 and keep (most of) 1701-1900,2001-2200
	assert.Equal(200, fixture.InsertBulk(2001, 2200, "/N/%d/Z", "/N/%d", ndn.CanBePrefixFlag))
	assert.Zero(fixture.FindBulk(1901, 2000, "/N/%d", ndn.CanBePrefixFlag))
	assert.True(fixture.FindBulk(1701, 1900, "/N/%d", ndn.CanBePrefixFlag) > 100)
}
