package fwdptest

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestSgTimer(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := intface.MustNew()
	face2 := intface.MustNew()
	collect2 := intface.Collect(face2)
	fixture.SetFibEntry("/A", "delay", face2.ID)

	// The strategy sets a 200ms timer, and then sends the Interest.
	// InterestLifetime is shorter than 200ms, so that strategy timer would not be triggered.
	face1.A.Tx() <- ndn.MakeInterest("/A/1", 100*time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(0, collect2.Count())
	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.GetFwdPit(i).ReadCounters().NEntries
	}))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(uint64(0), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.GetFwdPit(i).ReadCounters().NEntries
	}))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(0, collect2.Count())

	// InterestLifetime is longer than 200ms, and the strategy timer should be triggered.
	face1.A.Tx() <- ndn.MakeInterest("/A/2", 400*time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(0, collect2.Count())
	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.GetFwdPit(i).ReadCounters().NEntries
	}))
	time.Sleep(150 * time.Millisecond)
	assert.Equal(1, collect2.Count())
}
