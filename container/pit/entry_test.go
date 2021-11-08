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

	nonce1 := ndn.Nonce{0xA0, 0xA1, 0xA2, 0xA3}
	token1 := []byte{0xB0}
	interest1 := makeInterest("/A/B", 100*time.Millisecond, nonce1, setPitToken(token1), setFace(1001))

	nonce2 := ndn.Nonce{0xA4, 0xA5, 0xA6, 0xA7}
	token2 := []byte{0xB1, 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7, 0xB8, 0xB9, 0xBA, 0xBB, 0xBC, 0xBD, 0xBE, 0xBF, 0xB0, 0xB1, 0xB2}
	interest2 := makeInterest("/A/B", 400*time.Millisecond, nonce2, setPitToken(token2), setFace(1002))

	entry := fixture.Insert(interest1)
	require.NotNil(entry)
	assert.Len(entry.DnRecords(), 0)
	assert.NotNil(entry.InsertDnRecord(interest1))
	dnRecords := entry.DnRecords()
	assert.Len(dnRecords, 1)
	assert.Equal(token1, dnRecords[0].PitToken())
	assert.Equal(nonce1, dnRecords[0].Nonce())

	entry2 := fixture.Insert(interest2)
	require.NotNil(entry2)
	assert.Equal(uintptr(entry.Ptr()), uintptr(entry2.Ptr()))
	assert.NotNil(entry.InsertDnRecord(interest2))
	dnRecords = entry.DnRecords()
	assert.Len(dnRecords, 2)
	assert.Equal(token2, dnRecords[1].PitToken())
	assert.Equal(nonce2, dnRecords[1].Nonce())

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
		interest := makeInterest("/A/B", setFace(iface.ID(1000+i)))

		entry = fixture.Insert(interest)
		require.NotNil(entry)
		assert.NotNil(entry.InsertDnRecord(interest))
	}

	assert.Equal(1, fixture.Pit.Len())
	assert.Greater(fixture.CountMpInUse(), 1)

	fixture.Pit.Erase(entry)
	assert.Zero(fixture.Pit.Len())
	assert.Zero(fixture.CountMpInUse())
}

func TestEntryLongName(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(255)
	defer fixture.Close()

	names := []struct {
		Name string
		FH   string
	}{
		{strings.Repeat("/LLLLLLLL", 180), strings.Repeat("/FHFHFHFH", 70)},
		{strings.Repeat("/LLLLLLLL", 180), "/FH"},
	}
	entries := []*pit.Entry{}
	for i := 0; i < 4; i++ {
		name := names[i/2]
		interest := makeInterest(name.Name, ndn.MakeFHDelegation(1+i, name.FH),
			setActiveFwHint(0), setPitToken([]byte{0xB0, 0xB1, byte(i)}), setFace(iface.ID(1000+i)))

		entry := fixture.Insert(interest)
		require.NotNil(entry)
		entries = append(entries, entry)
	}
	assert.Equal(entries[0], entries[1])
	assert.Equal(entries[2], entries[3])

	assert.Equal(2, fixture.Pit.Len())
	assert.GreaterOrEqual(fixture.CountMpInUse(), 6)

	fixture.Pit.Erase(entries[0])
	fixture.Pit.Erase(entries[2])
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
	assert.NotNil(entry1.InsertDnRecord(interest1))
	assert.Equal(fibEntry1.FibSeqNum(), entry1.FibSeqNum())

	interest2 := makeInterest("/A/B")
	entry2, _ := fixture.Pit.Insert(interest2, fibEntry1)
	require.Equal(entry1, entry2)
	assert.Equal(fibEntry1.FibSeqNum(), entry2.FibSeqNum())

	fibEntry3 := fixture.InsertFibEntry("/A", 1003)
	assert.NotEqual(fibEntry1.FibSeqNum(), fibEntry3.FibSeqNum())
	entry3, _ := fixture.Pit.Insert(interest2, fibEntry3)
	require.Equal(entry2, entry3)
	assert.Equal(fibEntry3.FibSeqNum(), entry3.FibSeqNum())
}
