package cs_test

import (
	"testing"

	"ndn-dpdk/container/cs"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/ndn"
)

// Direct entries use ARC algorithm, but this only tests its LRU behavior.
func TestDirectLru(t *testing.T) {
	assert, _ := makeAR(t)
	var cfg pcct.Config
	cfg.CsCapMd = 400
	cfg.CsCapMi = 100
	fixture := NewFixture(cfg)
	defer fixture.Close()
	assert.Equal(400, fixture.Cs.GetCapacity(cs.CSL_MD))

	// insert 1-2000, should keep (most of) 1601-2000 direct entries
	assert.Equal(2000, fixture.InsertBulk(1, 2000, "/N/%d", "/N/%d", ndn.MustBeFreshFlag))
	assert.True(fixture.Cs.CountEntries(cs.CSL_MD) <= 400)
	assert.Zero(fixture.FindBulk(1, 1600, "/N/%d/Z", ndn.MustBeFreshFlag))

	// 1701-1900 become most-recently-used
	assert.Equal(200, fixture.FindBulk(1701, 1900, "/N/%d", ndn.MustBeFreshFlag))

	// insert 2001-2200, should evict 1901-2000 and keep (most of) 1701-1900,2001-2200
	assert.Equal(200, fixture.InsertBulk(2001, 2200, "/N/%d", "/N/%d", ndn.MustBeFreshFlag))
	assert.Zero(fixture.FindBulk(1901, 2000, "/N/%d", ndn.MustBeFreshFlag))
	assert.True(fixture.FindBulk(1701, 1900, "/N/%d", ndn.MustBeFreshFlag) > 100)
}

func TestIndirectLru(t *testing.T) {
	assert, _ := makeAR(t)
	var cfg pcct.Config
	cfg.CsCapMd = 600
	cfg.CsCapMi = 400
	fixture := NewFixture(cfg)
	defer fixture.Close()
	assert.Equal(400, fixture.Cs.GetCapacity(cs.CSL_MI))

	// insert 1-2000, should keep (most of) 1601-2000 indirect entries
	assert.Equal(2000, fixture.InsertBulk(1, 2000, "/N/%d/Z", "/N/%d", ndn.CanBePrefixFlag))
	assert.True(fixture.Cs.CountEntries(cs.CSL_MI) <= 400)
	assert.Zero(fixture.FindBulk(1, 1600, "/N/%d", ndn.CanBePrefixFlag))

	// 1701-1900 become most-recently-used
	assert.Equal(200, fixture.FindBulk(1701, 1900, "/N/%d", ndn.CanBePrefixFlag))

	// insert 2001-2200, should evict 1901-2000 and keep (most of) 1701-1900,2001-2200
	assert.Equal(200, fixture.InsertBulk(2001, 2200, "/N/%d/Z", "/N/%d", ndn.CanBePrefixFlag))
	assert.Zero(fixture.FindBulk(1901, 2000, "/N/%d", ndn.CanBePrefixFlag))
	assert.True(fixture.FindBulk(1701, 1900, "/N/%d", ndn.CanBePrefixFlag) > 100)
}
