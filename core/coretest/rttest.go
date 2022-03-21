package coretest

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
)

// RttEstimator represents an RTT estimator implementation.
type RttEstimator interface {
	Push(now time.Time, rtt time.Duration, withinRTT bool)
	Backoff()
	SRTT() time.Duration
	RTO() time.Duration
}

// RunRttEstimatorTest tests an RTT estimator implementation.
func RunRttEstimatorTest(t testing.TB, rtte RttEstimator) {
	assert, _ := testenv.MakeAR(t)
	durationDelta := float64(time.Millisecond)

	assert.InDelta(time.Second, rtte.RTO(), durationDelta)

	now := time.Now()
	rtte.Push(now, 500*time.Millisecond, false)
	// sRtt=500ms, rttVar=250ms
	assert.InDelta(500*time.Millisecond, rtte.SRTT(), durationDelta)
	assert.InDelta(1500*time.Millisecond, rtte.RTO(), durationDelta)

	now = now.Add(300 * time.Millisecond)
	rtte.Push(now, 800*time.Millisecond, true) // not collected within RTT
	assert.InDelta(500*time.Millisecond, rtte.SRTT(), durationDelta)
	assert.InDelta(1500*time.Millisecond, rtte.RTO(), durationDelta)

	now = now.Add(300 * time.Millisecond)
	rtte.Push(now, 800*time.Millisecond, false)
	// sRtt=537.5ms, rttVar=262.5ms
	assert.InDelta(537*time.Millisecond, rtte.SRTT(), durationDelta)
	assert.InDelta(1587*time.Millisecond, rtte.RTO(), durationDelta)

	rtte.Backoff()
	assert.InDelta(537*time.Millisecond, rtte.SRTT(), durationDelta)
	assert.InDelta(3175*time.Millisecond, rtte.RTO(), durationDelta)

	now = now.Add(800 * time.Millisecond)
	rtte.Push(now, 100*time.Millisecond, false)
	// sRtt=482.8175ms, rttVar=306.25ms
	assert.InDelta(482*time.Millisecond, rtte.SRTT(), durationDelta)
	assert.InDelta(1707*time.Millisecond, rtte.RTO(), durationDelta)

	rtte.Backoff()
	assert.InDelta(482*time.Millisecond, rtte.SRTT(), durationDelta)
	assert.InDelta(3415*time.Millisecond, rtte.RTO(), durationDelta)

	for i := 0; i < 20; i++ {
		rtte.Backoff()
	}
	assert.InDelta(482*time.Millisecond, rtte.SRTT(), durationDelta)
	assert.InDelta(60*time.Second, rtte.RTO(), durationDelta)

	for i := 0; i < 80; i++ {
		now = now.Add(800 * time.Millisecond)
		rtte.Push(now, 10*time.Millisecond, false)
	}
	assert.InDelta(10*time.Millisecond, rtte.SRTT(), durationDelta)
	assert.InDelta(200*time.Millisecond, rtte.RTO(), durationDelta)
}
