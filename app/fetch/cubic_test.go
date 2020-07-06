package fetch_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func TestCubic(t *testing.T) {
	assert, _ := makeAR(t)

	var ca fetch.Cubic
	ca.Init()

	assert.Equal(2, ca.Cwnd())

	now := eal.TscNow()
	rtt := 100 * time.Millisecond

	// slow start
	for i := 0; i < 98; i++ {
		ca.Increase(now, rtt)
		now = now.Add(5 * time.Millisecond)
	}
	assert.Equal(100, ca.Cwnd())

	// enter congestion avoidance
	ca.Decrease(now)
	assert.Equal(70, ca.Cwnd())
	now = now.Add(5 * time.Millisecond)

	// increase window
	firstCwnd := ca.Cwnd()
	lastCwnd := firstCwnd
	for i := 0; i < 1000; i++ {
		ca.Increase(now, rtt)
		thisCwnd := ca.Cwnd()
		assert.GreaterOrEqual(thisCwnd, lastCwnd)
		lastCwnd = thisCwnd
		now = now.Add(time.Millisecond)
	}
	assert.Greater(lastCwnd, firstCwnd)

	// decrease window
	ca.Decrease(now)
	thisCwnd := ca.Cwnd()
	assert.Less(thisCwnd, lastCwnd)
	now = now.Add(5 * time.Millisecond)
}
