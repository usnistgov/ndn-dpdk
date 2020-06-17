package pit_test

import (
	"strings"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndntestenv"
)

func TestEntryExpiry(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(255)
	defer fixture.Close()

	// lifetime 50ms
	interest1 := makeInterest("/A/B", 50*time.Millisecond)
	ndntestenv.SetPitToken(interest1, 0xB0B1B2B3B4B5B6B7)
	ndntestenv.SetPort(interest1, 1001)

	// lifetime 300ms
	interest2 := makeInterest("/A/B", 300*time.Millisecond)
	ndntestenv.SetPitToken(interest2, 0xB8B9BABBBCBDBEBF)
	ndntestenv.SetPort(interest2, 1002)

	entry := fixture.Insert(interest1)
	require.NotNil(entry)
	assert.Len(entry.ListDns(), 0)
	assert.NotNil(entry.InsertDn(interest1))
	assert.Len(entry.ListDns(), 1)

	entry2 := fixture.Insert(interest2)
	require.NotNil(entry2)
	assert.Equal(uintptr(entry.GetPtr()), uintptr(entry2.GetPtr()))
	assert.NotNil(entry.InsertDn(interest2))
	assert.Len(entry.ListDns(), 2)

	time.Sleep(100 * time.Millisecond)
	fixture.Pit.TriggerTimeoutSched()
	assert.Equal(1, fixture.Pit.Len())
	time.Sleep(250 * time.Millisecond)
	fixture.Pit.TriggerTimeoutSched()
	assert.Zero(fixture.Pit.Len())
}

func TestEntryExtend(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(255)
	defer fixture.Close()

	var entry *pit.Entry

	for i := 0; i < 512; i++ {
		interest := makeInterest("/A/B")
		ndntestenv.SetPitToken(interest, uint64(0xB0B1B2B300000000)|uint64(i))
		ndntestenv.SetPort(interest, uint16(1000+i))

		entry = fixture.Insert(interest)
		require.NotNil(entry)
		assert.NotNil(entry.InsertDn(interest))
	}

	assert.Equal(1, fixture.Pit.Len())
	assert.True(fixture.CountMpInUse() > 1)

	fixture.Pit.Erase(*entry)
	assert.Zero(fixture.Pit.Len())
	assert.Zero(fixture.CountMpInUse())
}

func TestEntryLongName(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(255)
	defer fixture.Close()

	interest := makeInterest(strings.Repeat("/LLLLLLLL", 180),
		ndni.FHDelegation{1, strings.Repeat("/FHFHFHFH", 70)},
		ndni.ActiveFHDelegation(0))
	ndntestenv.SetPitToken(interest, 0xB0B1B2B3B4B5B6B7)
	ndntestenv.SetPort(interest, 1000)

	entry := fixture.Insert(interest)
	require.NotNil(entry)

	assert.Equal(1, fixture.Pit.Len())
	assert.True(fixture.CountMpInUse() > 1)

	fixture.Pit.Erase(*entry)
	assert.Zero(fixture.Pit.Len())
	assert.Zero(fixture.CountMpInUse())
}

func TestEntryFibRef(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(255)
	defer fixture.Close()

	fibEntry1 := fixture.InsertFibEntry("/A", 1001)
	interest1 := makeInterest("/A/B")
	entry1, _ := fixture.Pit.Insert(interest1, fibEntry1)
	require.NotNil(entry1)
	assert.NotNil(entry1.InsertDn(interest1))
	assert.Equal(fibEntry1.GetSeqNum(), entry1.GetFibSeqNum())

	interest2 := makeInterest("/A/B")
	entry2, _ := fixture.Pit.Insert(interest2, fibEntry1)
	require.Equal(entry1, entry2)
	assert.Equal(fibEntry1.GetSeqNum(), entry2.GetFibSeqNum())

	fibEntry3 := fixture.InsertFibEntry("/A", 1003)
	assert.NotEqual(fibEntry1.GetSeqNum(), fibEntry3.GetSeqNum())
	entry3, _ := fixture.Pit.Insert(interest2, fibEntry3)
	require.Equal(entry2, entry3)
	assert.Equal(fibEntry3.GetSeqNum(), entry3.GetFibSeqNum())
}
