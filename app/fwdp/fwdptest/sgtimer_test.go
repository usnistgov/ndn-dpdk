package fwdptest

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
)

func TestSgTimer(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	fixture.SetFibEntry("/A", "delay", face2.GetFaceId())

	// The strategy sets a 200ms timer, and then sends the Interest.
	// InterestLifetime is shorter than 200ms, so that strategy timer would not be triggered.
	interest1 := makeInterest("/A/1", 100*time.Millisecond)
	face1.Rx(interest1)
	time.Sleep(50 * time.Millisecond)
	assert.Len(face2.TxInterests, 0)
	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.GetFwdPit(i).ReadCounters().NEntries
	}))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(uint64(0), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.GetFwdPit(i).ReadCounters().NEntries
	}))
	time.Sleep(100 * time.Millisecond)
	assert.Len(face2.TxInterests, 0)

	// InterestLifetime is longer than 200ms, and the strategy timer should be triggered.
	interest2 := makeInterest("/A/2", 400*time.Millisecond)
	face1.Rx(interest2)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face2.TxInterests, 0)
	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.GetFwdPit(i).ReadCounters().NEntries
	}))
	time.Sleep(150 * time.Millisecond)
	assert.Len(face2.TxInterests, 1)
}
