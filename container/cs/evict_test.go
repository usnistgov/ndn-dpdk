package cs_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/cs"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// Direct entries use ARC algorithm, but this only tests its LRU behavior.
func TestDirectLru(t *testing.T) {
	assert, _ := makeAR(t)
	var cfg pcct.Config
	cfg.CsCapMd = 400
	cfg.CsCapMi = 100
	fixture := NewFixture(cfg)
	defer fixture.Close()
	assert.Equal(400, fixture.Cs.Capacity(cs.CslMd))

	// insert 1-2000, should keep (most of) 1601-2000 direct entries
	assert.Equal(2000, fixture.InsertBulk(1, 2000, "/N/%d", "/N/%d", ndn.MustBeFreshFlag))
	assert.True(fixture.Cs.CountEntries(cs.CslMd) <= 400)
	assert.Zero(fixture.FindBulk(1, 1600, "/N/%d/Z", ndn.MustBeFreshFlag))

	// 1701-1900 become most-recently-used
	assert.Equal(200, fixture.FindBulk(1701, 1900, "/N/%d", ndn.MustBeFreshFlag))
	assert.Greater(fixture.Cs.CountEntries(cs.CslMdT2), 100)

	// insert 2001-2200, should evict 1901-2000 and keep (most of) 1701-1900,2001-2200
	assert.Equal(200, fixture.InsertBulk(2001, 2200, "/N/%d", "/N/%d", ndn.MustBeFreshFlag))
	assert.Zero(fixture.FindBulk(1901, 2000, "/N/%d", ndn.MustBeFreshFlag))
	assert.True(fixture.FindBulk(1701, 1900, "/N/%d", ndn.MustBeFreshFlag) > 100)
}

// This test partially verifies ARC list size updates.
func TestDirectArc(t *testing.T) {
	assert, _ := makeAR(t)
	var cfg pcct.Config
	cfg.CsCapMd = 100
	cfg.CsCapMi = 100
	fixture := NewFixture(cfg)
	defer fixture.Close()

	// insert 1-100 (NEW), p=0, T1=[1..100]
	fixture.InsertBulk(1, 100, "/N/%d", "/N/%d")
	assert.InDelta(0.0, fixture.Cs.ReadDirectArcP(), 0.1)
	assert.Equal(100, fixture.Cs.CountEntries(cs.CslMdT1))
	assert.Equal(0, fixture.Cs.CountEntries(cs.CslMdB1))
	assert.Equal(0, fixture.Cs.CountEntries(cs.CslMdT2))
	assert.Equal(0, fixture.Cs.CountEntries(cs.CslMdB2))

	// use 1-60 (T1), p=0, T1=[61..100], T2=[1..60]
	fixture.FindBulk(1, 60, "/N/%d", "/N/%d")
	assert.InDelta(0.0, fixture.Cs.ReadDirectArcP(), 0.1)
	assert.Equal(40, fixture.Cs.CountEntries(cs.CslMdT1))
	assert.Equal(0, fixture.Cs.CountEntries(cs.CslMdB1))
	assert.Equal(60, fixture.Cs.CountEntries(cs.CslMdT2))
	assert.Equal(0, fixture.Cs.CountEntries(cs.CslMdB2))

	// insert 101-130 (NEW), p=0, T1=[91..130], B1=[61..90], T2=[1..60]
	fixture.InsertBulk(101, 130, "/N/%d", "/N/%d")
	assert.InDelta(0.0, fixture.Cs.ReadDirectArcP(), 0.1)
	assert.Equal(40, fixture.Cs.CountEntries(cs.CslMdT1))
	assert.Equal(30, fixture.Cs.CountEntries(cs.CslMdB1))
	assert.Equal(60, fixture.Cs.CountEntries(cs.CslMdT2))
	assert.Equal(0, fixture.Cs.CountEntries(cs.CslMdB2))

	// insert 61-80 (B1), p=20, T1=[111..130], B1=[81..110], T2=[1..80]
	fixture.InsertBulk(61, 80, "/N/%d", "/N/%d")
	assert.InDelta(20.0, fixture.Cs.ReadDirectArcP(), 0.1)
	assert.Equal(20, fixture.Cs.CountEntries(cs.CslMdT1))
	assert.Equal(30, fixture.Cs.CountEntries(cs.CslMdB1))
	assert.Equal(80, fixture.Cs.CountEntries(cs.CslMdT2))
	assert.Equal(0, fixture.Cs.CountEntries(cs.CslMdB2))

	// insert 111-120 (T1), p=20, T1=[121..130], B1=[81..110], T2=[1..80,111..120]
	fixture.InsertBulk(111, 120, "/N/%d", "/N/%d")
	assert.InDelta(20.0, fixture.Cs.ReadDirectArcP(), 0.1)
	assert.Equal(10, fixture.Cs.CountEntries(cs.CslMdT1))
	assert.Equal(30, fixture.Cs.CountEntries(cs.CslMdB1))
	assert.Equal(90, fixture.Cs.CountEntries(cs.CslMdT2))
	assert.Equal(0, fixture.Cs.CountEntries(cs.CslMdB2))

	// insert 131-140 (NEW), p=20, T1=[121..140], B1=[81..110], T2=[11..80,111..120], B2=[1..10]
	fixture.InsertBulk(131, 140, "/N/%d", "/N/%d")
	assert.InDelta(20.0, fixture.Cs.ReadDirectArcP(), 0.1)
	assert.Equal(20, fixture.Cs.CountEntries(cs.CslMdT1))
	assert.Equal(30, fixture.Cs.CountEntries(cs.CslMdB1))
	assert.Equal(80, fixture.Cs.CountEntries(cs.CslMdT2))
	assert.Equal(10, fixture.Cs.CountEntries(cs.CslMdB2))

	// insert 1 (B2), p=20-30/10=17, T1=[122..140], B1=[81..110,121], T2=[11..80,111..120,1], B2=[2..10]
	fixture.InsertBulk(1, 1, "/N/%d", "/N/%d")
	assert.InDelta(17.0, fixture.Cs.ReadDirectArcP(), 0.1)
	assert.Equal(19, fixture.Cs.CountEntries(cs.CslMdT1))
	assert.Equal(31, fixture.Cs.CountEntries(cs.CslMdB1))
	assert.Equal(81, fixture.Cs.CountEntries(cs.CslMdT2))
	assert.Equal(9, fixture.Cs.CountEntries(cs.CslMdB2))
}

func TestIndirectLru(t *testing.T) {
	assert, _ := makeAR(t)
	var cfg pcct.Config
	cfg.CsCapMd = 600
	cfg.CsCapMi = 400
	fixture := NewFixture(cfg)
	defer fixture.Close()
	assert.Equal(400, fixture.Cs.Capacity(cs.CslMi))

	// insert 1-2000, should keep (most of) 1601-2000 indirect entries
	assert.Equal(2000, fixture.InsertBulk(1, 2000, "/N/%d/Z", "/N/%d", ndn.CanBePrefixFlag))
	assert.True(fixture.Cs.CountEntries(cs.CslMi) <= 400)
	assert.Zero(fixture.FindBulk(1, 1600, "/N/%d", ndn.CanBePrefixFlag))

	// 1701-1900 become most-recently-used
	assert.Equal(200, fixture.FindBulk(1701, 1900, "/N/%d", ndn.CanBePrefixFlag))

	// insert 2001-2200, should evict 1901-2000 and keep (most of) 1701-1900,2001-2200
	assert.Equal(200, fixture.InsertBulk(2001, 2200, "/N/%d/Z", "/N/%d", ndn.CanBePrefixFlag))
	assert.Zero(fixture.FindBulk(1901, 2000, "/N/%d", ndn.CanBePrefixFlag))
	assert.True(fixture.FindBulk(1701, 1900, "/N/%d", ndn.CanBePrefixFlag) > 100)
}
