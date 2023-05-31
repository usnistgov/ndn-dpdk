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
	fixture := NewFixture(t, pcct.Config{
		CsMemoryCapacity:   400,
		CsIndirectCapacity: 100,
	})
	assert.Equal(400, fixture.Cs.Capacity(cs.ListDirect))

	// insert 1-2000, should keep (most of) 1601-2000 direct entries
	assert.Equal(2000, fixture.InsertBulk(1, 2000, "/N/%d", "/N/%d", ndn.MustBeFreshFlag))
	assert.LessOrEqual(fixture.Cs.CountEntries(cs.ListDirect), 400)
	assert.Zero(fixture.FindBulk(1, 1600, "/N/%d/Z", ndn.MustBeFreshFlag))

	// 1701-1900 become most-recently-used
	assert.Equal(200, fixture.FindBulk(1701, 1900, "/N/%d", ndn.MustBeFreshFlag))
	assert.Greater(fixture.Cs.CountEntries(cs.ListDirectT2), 100)

	// insert 2001-2200, should evict 1901-2000 and keep (most of) 1701-1900,2001-2200
	assert.Equal(200, fixture.InsertBulk(2001, 2200, "/N/%d", "/N/%d", ndn.MustBeFreshFlag))
	assert.Zero(fixture.FindBulk(1901, 2000, "/N/%d", ndn.MustBeFreshFlag))
	assert.True(fixture.FindBulk(1701, 1900, "/N/%d", ndn.MustBeFreshFlag) > 100)
}

// This test partially verifies ARC list size updates.
func TestDirectArc(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t, pcct.Config{
		CsMemoryCapacity:   100,
		CsIndirectCapacity: 100,
	})

	// insert 1..100 (NEW), p=0, T1=[1..100]
	fixture.InsertBulk(1, 100, "/N/%d", "/N/%d")
	assert.InDelta(0.00, fixture.Cs.ReadDirectArcP(), 0.01)
	assert.Equal(100, fixture.Cs.CountEntries(cs.ListDirectT1))
	assert.Equal(0, fixture.Cs.CountEntries(cs.ListDirectB1))
	assert.Equal(0, fixture.Cs.CountEntries(cs.ListDirectT2))
	assert.Equal(0, fixture.Cs.CountEntries(cs.ListDirectB2))

	// use 1..60 (T1), p=0, T1=[61..100], T2=[1..60]
	fixture.FindBulk(1, 60, "/N/%d", "/N/%d")
	assert.InDelta(0.00, fixture.Cs.ReadDirectArcP(), 0.01)
	assert.Equal(40, fixture.Cs.CountEntries(cs.ListDirectT1))
	assert.Equal(0, fixture.Cs.CountEntries(cs.ListDirectB1))
	assert.Equal(60, fixture.Cs.CountEntries(cs.ListDirectT2))
	assert.Equal(0, fixture.Cs.CountEntries(cs.ListDirectB2))

	// use 41..60 (T2), p=0, T1=[61..100], T2=[1..60]
	fixture.FindBulk(41, 60, "/N/%d", "/N/%d")
	assert.InDelta(0.00, fixture.Cs.ReadDirectArcP(), 0.01)
	assert.Equal(40, fixture.Cs.CountEntries(cs.ListDirectT1))
	assert.Equal(0, fixture.Cs.CountEntries(cs.ListDirectB1))
	assert.Equal(60, fixture.Cs.CountEntries(cs.ListDirectT2))
	assert.Equal(0, fixture.Cs.CountEntries(cs.ListDirectB2))

	// insert 101..130 (NEW), p=0, T1=[91..130], B1=[61..90], T2=[1..60]
	fixture.InsertBulk(101, 130, "/N/%d", "/N/%d")
	assert.InDelta(0.00, fixture.Cs.ReadDirectArcP(), 0.01)
	assert.Equal(40, fixture.Cs.CountEntries(cs.ListDirectT1))
	assert.Equal(30, fixture.Cs.CountEntries(cs.ListDirectB1))
	assert.Equal(60, fixture.Cs.CountEntries(cs.ListDirectT2))
	assert.Equal(0, fixture.Cs.CountEntries(cs.ListDirectB2))

	// insert 61..80 (B1), p=20, T1=[111..130], B1=[81..110], T2=[1..80]
	fixture.InsertBulk(61, 80, "/N/%d", "/N/%d")
	assert.InDelta(20.00, fixture.Cs.ReadDirectArcP(), 0.01)
	assert.Equal(20, fixture.Cs.CountEntries(cs.ListDirectT1))
	assert.Equal(30, fixture.Cs.CountEntries(cs.ListDirectB1))
	assert.Equal(80, fixture.Cs.CountEntries(cs.ListDirectT2))
	assert.Equal(0, fixture.Cs.CountEntries(cs.ListDirectB2))

	// insert 111..120 (T1), p=20, T1=[121..130], B1=[81..110], T2=[1..80,111..120]
	fixture.InsertBulk(111, 120, "/N/%d", "/N/%d")
	assert.InDelta(20.00, fixture.Cs.ReadDirectArcP(), 0.01)
	assert.Equal(10, fixture.Cs.CountEntries(cs.ListDirectT1))
	assert.Equal(30, fixture.Cs.CountEntries(cs.ListDirectB1))
	assert.Equal(90, fixture.Cs.CountEntries(cs.ListDirectT2))
	assert.Equal(0, fixture.Cs.CountEntries(cs.ListDirectB2))

	// insert 131..140 (NEW), p=20, T1=[121..140], B1=[81..110], T2=[11..80,111..120], B2=[1..10]
	fixture.InsertBulk(131, 140, "/N/%d", "/N/%d")
	assert.InDelta(20.0, fixture.Cs.ReadDirectArcP(), 0.01)
	assert.Equal(20, fixture.Cs.CountEntries(cs.ListDirectT1))
	assert.Equal(30, fixture.Cs.CountEntries(cs.ListDirectB1))
	assert.Equal(80, fixture.Cs.CountEntries(cs.ListDirectT2))
	assert.Equal(10, fixture.Cs.CountEntries(cs.ListDirectB2))

	// insert 1 (B2), p=20.00-30/10=17.00, T1=[122..140], B1=[81..110,121], T2=[11..80,111..120,1], B2=[2..10]
	// insert 2 (B2), p=17.00-31/9=13.56, T1=[123..140], B1=[81..110,121..122], T2=[11..80,111..120,1..2], B2=[3..10]
	fixture.InsertBulk(1, 2, "/N/%d", "/N/%d")
	assert.InDelta(13.56, fixture.Cs.ReadDirectArcP(), 0.01)
	assert.Equal(18, fixture.Cs.CountEntries(cs.ListDirectT1))
	assert.Equal(32, fixture.Cs.CountEntries(cs.ListDirectB1))
	assert.Equal(82, fixture.Cs.CountEntries(cs.ListDirectT2))
	assert.Equal(8, fixture.Cs.CountEntries(cs.ListDirectB2))
}

func TestIndirectLru(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t, pcct.Config{
		CsMemoryCapacity:   600,
		CsIndirectCapacity: 400,
	})
	assert.Equal(400, fixture.Cs.Capacity(cs.ListIndirect))

	// insert 1..2000, should keep (most of) 1601..2000 indirect entries
	assert.Equal(2000, fixture.InsertBulk(1, 2000, "/N/%d/Z", "/N/%d", ndn.CanBePrefixFlag))
	assert.LessOrEqual(fixture.Cs.CountEntries(cs.ListIndirect), 400)
	assert.Zero(fixture.FindBulk(1, 1600, "/N/%d", ndn.CanBePrefixFlag))

	// 1701..1900 become most-recently-used
	assert.Equal(200, fixture.FindBulk(1701, 1900, "/N/%d", ndn.CanBePrefixFlag))

	// insert 2001..2200, should evict 1901..2000 and keep (most of) 1701..1900,2001..2200
	assert.Equal(200, fixture.InsertBulk(2001, 2200, "/N/%d/Z", "/N/%d", ndn.CanBePrefixFlag))
	assert.Zero(fixture.FindBulk(1901, 2000, "/N/%d", ndn.CanBePrefixFlag))
	assert.Greater(fixture.FindBulk(1701, 1900, "/N/%d", ndn.CanBePrefixFlag), 100)
}

func TestIndirectDep(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t, pcct.Config{
		CsMemoryCapacity:   100,
		CsIndirectCapacity: 200,
	})

	assert.Equal(200, fixture.InsertBulk(1, 200, "/N/%d/Z/Z", "/N/%d", ndn.CanBePrefixFlag))
	assert.GreaterOrEqual(fixture.Cs.CountEntries(cs.ListIndirect), 100)
	assert.Equal(50, fixture.InsertBulk(151, 200, "/N/%d/Z/Z", "/N/%d/Z", ndn.CanBePrefixFlag))
	// T1=[101..200], INDIRECT=[101..200,151..200Z]
	assert.GreaterOrEqual(fixture.Cs.CountEntries(cs.ListDirect), 100)
	assert.GreaterOrEqual(fixture.Cs.CountEntries(cs.ListIndirect), 150)

	assert.Equal(100, fixture.FindBulk(101, 200, "/N/%d/Z/Z"))
	assert.Equal(0, fixture.FindBulk(101, 150, "/N/%d/Z", ndn.CanBePrefixFlag))
	assert.Equal(50, fixture.FindBulk(151, 200, "/N/%d/Z", ndn.CanBePrefixFlag))
	assert.Equal(100, fixture.FindBulk(101, 200, "/N/%d", ndn.CanBePrefixFlag))
	// T2=[101..200], INDIRECT=[101..200,151..200Z]
	assert.GreaterOrEqual(fixture.Cs.CountEntries(cs.ListDirect), 100)
	assert.GreaterOrEqual(fixture.Cs.CountEntries(cs.ListIndirect), 150)

	assert.Equal(200, fixture.InsertBulk(201, 400, "/N/%d/Z/Z", "/N/%d/Z/Z"))
	assert.Equal(110, fixture.InsertBulk(291, 400, "/N/%d/Z/Z", "/N/%d/Z/Z"))
	// T2=[301..400], INDIRECT=[]
	assert.GreaterOrEqual(fixture.Cs.CountEntries(cs.ListDirect), 100)
	assert.GreaterOrEqual(fixture.Cs.CountEntries(cs.ListIndirect), 0)
}
