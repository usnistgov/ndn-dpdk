package pit_test

import (
	"strings"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestEntryExpiry(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(255)
	defer fixture.Close()

	// lifetime 100ms
	interest1 := makeInterest("/A/B", 100*time.Millisecond, setPitToken(0xB0B1B2B3B4B5B6B7), setFace(1001))

	// lifetime 400ms
	interest2 := makeInterest("/A/B", 400*time.Millisecond, setPitToken(0xB8B9BABBBCBDBEBF), setFace(1002))

	entry := fixture.Insert(interest1)
	require.NotNil(entry)
	assert.Len(entry.ListDns(), 0)
	assert.NotNil(entry.InsertDn(interest1))
	assert.Len(entry.ListDns(), 1)

	entry2 := fixture.Insert(interest2)
	require.NotNil(entry2)
	assert.Equal(uintptr(entry.Ptr()), uintptr(entry2.Ptr()))
	assert.NotNil(entry.InsertDn(interest2))
	assert.Len(entry.ListDns(), 2)

	time.Sleep(200 * time.Millisecond)
	fixture.Pit.TriggerTimeoutSched()
	assert.Equal(1, fixture.Pit.Len())
	time.Sleep(400 * time.Millisecond)
	fixture.Pit.TriggerTimeoutSched()
	assert.Zero(fixture.Pit.Len())
}

func TestEntryExtend(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(255)
	defer fixture.Close()

	var entry *pit.Entry

	for i := 0; i < 512; i++ {
		interest := makeInterest("/A/B", setPitToken(0xB0B1B2B300000000|uint64(i)), setFace(iface.ID(1000+i)))

		entry = fixture.Insert(interest)
		require.NotNil(entry)
		assert.NotNil(entry.InsertDn(interest))
	}

	assert.Equal(1, fixture.Pit.Len())
	assert.True(fixture.CountMpInUse() > 1)

	fixture.Pit.Erase(entry)
	assert.Zero(fixture.Pit.Len())
	assert.Zero(fixture.CountMpInUse())
}

func TestEntryLongName(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(255)
	defer fixture.Close()

	interest := makeInterest(strings.Repeat("/LLLLLLLL", 180),
		ndn.MakeFHDelegation(1, strings.Repeat("/FHFHFHFH", 70)),
		setActiveFwHint(0), setPitToken(0xB0B1B2B3B4B5B6B7), setFace(1000))

	entry := fixture.Insert(interest)
	require.NotNil(entry)

	assert.Equal(1, fixture.Pit.Len())
	assert.True(fixture.CountMpInUse() > 1)

	fixture.Pit.Erase(entry)
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
	assert.Equal(fibEntry1.FibSeqNum(), entry1.FibFibSeqNum())

	interest2 := makeInterest("/A/B")
	entry2, _ := fixture.Pit.Insert(interest2, fibEntry1)
	require.Equal(entry1, entry2)
	assert.Equal(fibEntry1.FibSeqNum(), entry2.FibFibSeqNum())

	fibEntry3 := fixture.InsertFibEntry("/A", 1003)
	assert.NotEqual(fibEntry1.FibSeqNum(), fibEntry3.FibSeqNum())
	entry3, _ := fixture.Pit.Insert(interest2, fibEntry3)
	require.Equal(entry2, entry3)
	assert.Equal(fibEntry3.FibSeqNum(), entry3.FibFibSeqNum())
}
