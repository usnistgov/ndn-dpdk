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

	face1, face2 := intface.MustNew(), intface.MustNew()
	collect2 := intface.Collect(face2)
	fixture.SetFibEntryParams("/A", "delay", map[string]interface{}{"delay": 200}, face2.ID)

	// The strategy sets a 200ms timer, and then sends the Interest.
	// InterestLifetime is shorter than 200ms, so that strategy timer would not be triggered.
	face1.A.Tx() <- ndn.MakeInterest("/A/1", 100*time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(0, collect2.Count())
	assert.Equal(uint64(1), fixture.SumCounter(func(fwd *fwdp.Fwd) uint64 {
		return fwd.Pit().Counters().NEntries
	}))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(uint64(0), fixture.SumCounter(func(fwd *fwdp.Fwd) uint64 {
		return fwd.Pit().Counters().NEntries
	}))
	time.Sleep(100 * time.Millisecond)
	assert.Equal(0, collect2.Count())

	// InterestLifetime is longer than 200ms, and the strategy timer should be triggered.
	face1.A.Tx() <- ndn.MakeInterest("/A/2", 400*time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(0, collect2.Count())
	assert.Equal(uint64(1), fixture.SumCounter(func(fwd *fwdp.Fwd) uint64 {
		return fwd.Pit().Counters().NEntries
	}))
	time.Sleep(150 * time.Millisecond)
	assert.Equal(1, collect2.Count())

	// Changing delay to 500ms.
	fixture.SetFibEntryParams("/A", "delay", map[string]interface{}{"delay": 500}, face2.ID)
	face1.A.Tx() <- ndn.MakeInterest("/A/2", 1000*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	assert.Equal(1, collect2.Count())
	time.Sleep(400 * time.Millisecond)
	assert.Equal(2, collect2.Count())
}
