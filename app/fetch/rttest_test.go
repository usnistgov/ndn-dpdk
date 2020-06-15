package fetch_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func TestRttEst(t *testing.T) {
	assert, _ := makeAR(t)
	durationDelta := float64(time.Millisecond)

	var rtte fetch.RttEst
	rtte.Init()
	assert.InDelta(time.Second, rtte.GetRto(), durationDelta)

	now := eal.TscNow()
	rtte.Push(now, 500*time.Millisecond)
	// sRtt=500ms, rttVar=250ms
	assert.InDelta(500*time.Millisecond, rtte.GetSRtt(), durationDelta)
	assert.InDelta(1500*time.Millisecond, rtte.GetRto(), durationDelta)

	now = now.Add(300 * time.Millisecond)
	rtte.Push(now, 800*time.Millisecond) // not collected within RTT
	assert.InDelta(500*time.Millisecond, rtte.GetSRtt(), durationDelta)
	assert.InDelta(1500*time.Millisecond, rtte.GetRto(), durationDelta)

	now = now.Add(300 * time.Millisecond)
	rtte.Push(now, 800*time.Millisecond)
	// sRtt=537.5ms, rttVar=262.5ms
	assert.InDelta(537*time.Millisecond, rtte.GetSRtt(), durationDelta)
	assert.InDelta(1587*time.Millisecond, rtte.GetRto(), durationDelta)

	rtte.Backoff()
	assert.InDelta(537*time.Millisecond, rtte.GetSRtt(), durationDelta)
	assert.InDelta(3175*time.Millisecond, rtte.GetRto(), durationDelta)

	now = now.Add(800 * time.Millisecond)
	rtte.Push(now, 100*time.Millisecond)
	// sRtt=482.8175ms, rttVar=306.25ms
	assert.InDelta(482*time.Millisecond, rtte.GetSRtt(), durationDelta)
	assert.InDelta(1707*time.Millisecond, rtte.GetRto(), durationDelta)

	rtte.Backoff()
	assert.InDelta(482*time.Millisecond, rtte.GetSRtt(), durationDelta)
	assert.InDelta(3415*time.Millisecond, rtte.GetRto(), durationDelta)

	for i := 0; i < 20; i++ {
		rtte.Backoff()
	}
	assert.InDelta(482*time.Millisecond, rtte.GetSRtt(), durationDelta)
	assert.InDelta(60*time.Second, rtte.GetRto(), durationDelta)

	for i := 0; i < 80; i++ {
		now = now.Add(800 * time.Millisecond)
		rtte.Push(now, 10*time.Millisecond)
	}
	assert.InDelta(10*time.Millisecond, rtte.GetSRtt(), durationDelta)
	assert.InDelta(200*time.Millisecond, rtte.GetRto(), durationDelta)
}
