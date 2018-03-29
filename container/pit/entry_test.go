package pit_test

import (
	"testing"
	"time"

	"ndn-dpdk/ndn/ndntestutil"
)

func TestEntry(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(255)
	defer fixture.Close()

	// lifetime 50ms
	interest1 := ndntestutil.MakeInterest("/A/B", 50*time.Millisecond)
	ndntestutil.SetPitToken(interest1, 0xB0B1B2B3B4B5B6B7)
	ndntestutil.SetFaceId(interest1, 1001)

	// lifetime 200ms
	interest2 := ndntestutil.MakeInterest("/A/B", 200*time.Millisecond)
	ndntestutil.SetPitToken(interest2, 0xB8B9BABBBCBDBEBF)
	ndntestutil.SetFaceId(interest2, 1002)

	entry := fixture.Insert(interest1)
	require.NotNil(entry)
	assert.Len(entry.ListDns(), 0)
	assert.True(entry.DnRxInterest(interest1))
	assert.Len(entry.ListDns(), 1)

	entry2 := fixture.Insert(interest2)
	require.NotNil(entry2)
	assert.Equal(uintptr(entry.GetPtr()), uintptr(entry2.GetPtr()))
	assert.True(entry.DnRxInterest(interest2))
	assert.Len(entry.ListDns(), 2)

	time.Sleep(100 * time.Millisecond)
	fixture.Pit.TriggerTimeoutSched()
	assert.Equal(1, fixture.Pit.Len())
	time.Sleep(150 * time.Millisecond)
	fixture.Pit.TriggerTimeoutSched()
	assert.Zero(fixture.Pit.Len())
}
