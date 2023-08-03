package cs_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/container/cs"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// C.Cs_PutIndirect "refresh indirect entry"
func TestIndirectRefresh(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t, pcct.Config{})

	assert.True(fixture.Insert(makeInterest("/A/1", ndn.CanBePrefixFlag), makeData("/A/1/P")))
	assert.Equal(1, fixture.Cs.CountEntries(cs.ListDirect))
	assert.Equal(1, fixture.Cs.CountEntries(cs.ListIndirect))

	assert.True(fixture.Insert(makeInterest("/A/1", ndn.CanBePrefixFlag, ndn.MustBeFreshFlag), makeData("/A/1/Q")))
	assert.Equal(2, fixture.Cs.CountEntries(cs.ListDirect))
	assert.Equal(1, fixture.Cs.CountEntries(cs.ListIndirect))

	if found := fixture.Find(makeInterest("/A/1", ndn.CanBePrefixFlag)); assert.NotNil(found) {
		nameEqual(assert, "/A/1/Q", found.Data().ToNPacket().Data)
	}
}

// C.Cs_PutIndirect "don't overwrite direct entry that has dependencies"
func TestIndirectNoOverwrite(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t, pcct.Config{})

	assert.True(fixture.Insert(makeInterest("/A", ndn.CanBePrefixFlag), makeData("/A/1")))
	assert.Equal(1, fixture.Cs.CountEntries(cs.ListDirect))
	assert.Equal(1, fixture.Cs.CountEntries(cs.ListIndirect))

	assert.True(fixture.Insert(makeInterest("/A/1", ndn.CanBePrefixFlag, ndn.MustBeFreshFlag), makeData("/A/1/Q", time.Hour)))
	assert.Equal(2, fixture.Cs.CountEntries(cs.ListDirect))
	assert.Equal(1, fixture.Cs.CountEntries(cs.ListIndirect))

	if found := fixture.Find(makeInterest("/A", ndn.CanBePrefixFlag)); assert.NotNil(found) {
		nameEqual(assert, "/A/1", found.Data().ToNPacket().Data)
	}
	if found := fixture.Find(makeInterest("/A/1", ndn.CanBePrefixFlag)); assert.NotNil(found) {
		nameEqual(assert, "/A/1", found.Data().ToNPacket().Data)
	}
	if found := fixture.Find(makeInterest("/A/1/Q", ndn.MustBeFreshFlag)); assert.NotNil(found) {
		nameEqual(assert, "/A/1/Q", found.Data().ToNPacket().Data)
	}
	assert.Nil(fixture.Find(makeInterest("/A/1", ndn.CanBePrefixFlag, ndn.MustBeFreshFlag)))
}

// C.Cs_PutIndirect "change direct entry to indirect entry"
func TestIndirectOverwrite(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t, pcct.Config{})

	assert.True(fixture.Insert(makeInterest("/A/1"), makeData("/A/1")))
	assert.Equal(1, fixture.Cs.CountEntries(cs.ListDirect))
	assert.Equal(0, fixture.Cs.CountEntries(cs.ListIndirect))

	assert.True(fixture.Insert(makeInterest("/A/1", ndn.CanBePrefixFlag, ndn.MustBeFreshFlag), makeData("/A/1/Q", time.Hour)))
	assert.Equal(1, fixture.Cs.CountEntries(cs.ListDirect))
	assert.Equal(1, fixture.Cs.CountEntries(cs.ListIndirect))

	assert.Nil(fixture.Find(makeInterest("/A/1")))
	if found := fixture.Find(makeInterest("/A/1", ndn.CanBePrefixFlag, ndn.MustBeFreshFlag)); assert.NotNil(found) {
		nameEqual(assert, "/A/1/Q", found.Data().ToNPacket().Data)
	}
}

// C.Cs_PutIndirect "indirect assoc err" (too many indirect entries associated with same direct entry)
func TestIndirectAssocErr(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t, pcct.Config{})

	for i, interestName := range []string{"/A/B/C/D/E/F", "/A/B/C/D/E", "/A/B/C/D", "/A/B/C"} {
		assert.True(fixture.Insert(makeInterest(interestName, ndn.CanBePrefixFlag, ndn.MustBeFreshFlag), makeData("/A/B/C/D/E/F/G")))
		assert.Equal(1, fixture.Cs.CountEntries(cs.ListDirect))
		assert.Equal(1+i, fixture.Cs.CountEntries(cs.ListIndirect))
		assert.Equal(2+i, fixture.CountMpInUse())
	}

	// with PCC erase
	assert.True(fixture.Insert(makeInterest("/A/B", ndn.CanBePrefixFlag, ndn.MustBeFreshFlag), makeData("/A/B/C/D/E/F/G")))
	assert.Equal(1, fixture.Cs.CountEntries(cs.ListDirect))
	assert.Equal(4, fixture.Cs.CountEntries(cs.ListIndirect))
	assert.Equal(5, fixture.CountMpInUse())

	// without PCC erase
	pitEntry1, csEntry1 := fixture.Pit.Insert(makeInterest("/A"), fixture.FibEntry)
	assert.NotNil(pitEntry1)
	assert.Nil(csEntry1)
	assert.Equal(6, fixture.CountMpInUse())
	assert.True(fixture.Insert(makeInterest("/A", ndn.CanBePrefixFlag, ndn.MustBeFreshFlag), makeData("/A/B/C/D/E/F/G")))
	assert.Equal(1, fixture.Cs.CountEntries(cs.ListDirect))
	assert.Equal(4, fixture.Cs.CountEntries(cs.ListIndirect))
	assert.Equal(6, fixture.CountMpInUse())
}
