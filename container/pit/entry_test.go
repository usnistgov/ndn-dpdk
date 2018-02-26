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
	defer pit.Close()

	// lifetime 50ms
	interest1 := ndntestutil.MakeInterest("641F " +
		"token=6208B0B1B2B3B4B5B6B7 payload=5013 " +
		"0511 name=0706 080141 080142 nonce=0A04A0A1A2A3 lifetime=0C0132")
	ndntestutil.SetFaceId(interest1, 1001)

	// lifetime 200ms
	interest2 := ndntestutil.MakeInterest("641F " +
		"token=6208B8B9BABBBCBDBEBF payload=5013 " +
		"0511 name=0706 080141 080142 nonce=0A04A8A9AAAB lifetime=0C01C8")
	ndntestutil.SetFaceId(interest2, 1002)

	entry, _ := pit.Insert(interest1)
	require.NotNil(entry)
	entry2, _ := pit.Insert(interest2)
	require.NotNil(entry2)
	assert.True(entry.SameAs(*entry2))

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
