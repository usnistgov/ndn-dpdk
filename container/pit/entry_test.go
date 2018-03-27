package pit_test

import (
	"testing"
	"time"

	"ndn-dpdk/ndn/ndntestutil"
)

func TestEntry(t *testing.T) {
	assert, require := makeAR(t)

	pit := createPit()
	defer pit.Pcct.Close()

	// lifetime 50ms
	interest1 := ndntestutil.MakeInterest("/A/B", 50*time.Millisecond)
	ndntestutil.SetPitToken(interest1, 0xB0B1B2B3B4B5B6B7)
	ndntestutil.SetFaceId(interest1, 1001)

	// lifetime 200ms
	interest2 := ndntestutil.MakeInterest("/A/B", 200*time.Millisecond)
	ndntestutil.SetPitToken(interest2, 0xB8B9BABBBCBDBEBF)
	ndntestutil.SetFaceId(interest2, 1002)

	entry, _ := pit.Insert(interest1)
	require.NotNil(entry)
	entry2, _ := pit.Insert(interest2)
	require.NotNil(entry2)
	assert.Equal(uintptr(entry.GetPtr()), uintptr(entry2.GetPtr()))

	assert.Len(entry.ListDns(), 0)

	assert.True(entry.DnRxInterest(interest1))
	assert.Len(entry.ListDns(), 1)

	assert.True(entry.DnRxInterest(interest2))
	assert.Len(entry.ListDns(), 2)

	time.Sleep(100 * time.Millisecond)
	pit.TriggerTimeoutSched()
	assert.Equal(1, pit.Len())
	time.Sleep(150 * time.Millisecond)
	pit.TriggerTimeoutSched()
	assert.Equal(0, pit.Len())
}
