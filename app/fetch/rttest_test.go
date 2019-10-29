package fetch_test

import (
	"testing"
	"time"

	"ndn-dpdk/app/fetch"
	"ndn-dpdk/dpdk"
)

func TestRttEst(t *testing.T) {
	assert, _ := makeAR(t)
	durationDelta := float64(dpdk.ToTscDuration(time.Millisecond))

	rtte := fetch.NewRttEst()
	assert.InDelta(dpdk.ToTscDuration(time.Second), rtte.GetRto(), durationDelta)

	now := dpdk.TscNow()
	since := now.Add(-500 * time.Millisecond)
	rtte.Push(since, now)
	// sRtt=500ms, rttVar=250ms
	assert.InDelta(dpdk.ToTscDuration(500*time.Millisecond), rtte.GetRtt(), durationDelta)
	assert.InDelta(dpdk.ToTscDuration(1500*time.Millisecond), rtte.GetRto(), durationDelta)

	now = now.Add(300 * time.Millisecond)
	since = now.Add(-800 * time.Millisecond)
	rtte.Push(since, now) // not collected within RTT
	assert.InDelta(dpdk.ToTscDuration(500*time.Millisecond), rtte.GetRtt(), durationDelta)
	assert.InDelta(dpdk.ToTscDuration(1500*time.Millisecond), rtte.GetRto(), durationDelta)

	now = now.Add(300 * time.Millisecond)
	since = now.Add(-800 * time.Millisecond)
	rtte.Push(since, now)
	// sRtt=537.5ms, rttVar=262.5ms
	assert.InDelta(dpdk.ToTscDuration(537500*time.Microsecond), rtte.GetRtt(), durationDelta)
	assert.InDelta(dpdk.ToTscDuration(1587500*time.Microsecond), rtte.GetRto(), durationDelta)

	rtte.Backoff()
	assert.InDelta(dpdk.ToTscDuration(537500*time.Microsecond), rtte.GetRtt(), durationDelta)
	assert.InDelta(dpdk.ToTscDuration(3175*time.Millisecond), rtte.GetRto(), durationDelta)

	now = now.Add(800 * time.Millisecond)
	since = now.Add(-100 * time.Millisecond)
	rtte.Push(since, now)
	// sRtt=482.8175ms, rttVar=306.25ms
	assert.InDelta(dpdk.ToTscDuration(482817*time.Microsecond), rtte.GetRtt(), durationDelta)
	assert.InDelta(dpdk.ToTscDuration(1707817*time.Microsecond), rtte.GetRto(), durationDelta)

	rtte.Backoff()
	assert.InDelta(dpdk.ToTscDuration(482817*time.Microsecond), rtte.GetRtt(), durationDelta)
	assert.InDelta(dpdk.ToTscDuration(3415635*time.Microsecond), rtte.GetRto(), durationDelta)

	for i := 0; i < 20; i++ {
		rtte.Backoff()
	}
	assert.InDelta(dpdk.ToTscDuration(482817*time.Microsecond), rtte.GetRtt(), durationDelta)
	assert.InDelta(dpdk.ToTscDuration(60*time.Second), rtte.GetRto(), durationDelta)

	for i := 0; i < 80; i++ {
		now = now.Add(800 * time.Millisecond)
		since = now.Add(-10 * time.Millisecond)
		rtte.Push(since, now)
	}
	assert.InDelta(dpdk.ToTscDuration(10*time.Millisecond), rtte.GetRtt(), durationDelta)
	assert.InDelta(dpdk.ToTscDuration(200*time.Millisecond), rtte.GetRto(), durationDelta)
}
