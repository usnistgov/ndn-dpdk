package fetchtest

/*
#include "../../../csrc/fetch/tcpcubic.h"
*/
import "C"
import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func ctestTcpCubic(t *testing.T) {
	assert, _ := makeAR(t)

	ca := &C.TcpCubic{}
	C.TcpCubic_Init(ca)

	now := eal.TscNow()
	rtt := 100 * time.Millisecond

	advance := func(d time.Duration) { now = now.Add(d) }
	cwnd := func() int { return int(C.TcpCubic_GetCwnd(ca)) }
	increase := func() { C.TcpCubic_Increase(ca, C.TscTime(now), C.double(eal.ToTscDuration(rtt))) }
	decrease := func() { C.TcpCubic_Decrease(ca, C.TscTime(now)) }

	assert.Equal(2, cwnd())

	// slow start
	for i := 0; i < 98; i++ {
		increase()
		advance(5 * time.Millisecond)
	}
	assert.Equal(100, cwnd())

	// enter congestion avoidance
	decrease()
	assert.Equal(70, cwnd())
	advance(5 * time.Millisecond)

	// increase window
	firstCwnd := cwnd()
	lastCwnd := firstCwnd
	for i := 0; i < 1000; i++ {
		increase()
		thisCwnd := cwnd()
		assert.GreaterOrEqual(thisCwnd, lastCwnd)
		lastCwnd = thisCwnd
		advance(time.Millisecond)
	}
	assert.Greater(lastCwnd, firstCwnd)

	// decrease window
	decrease()
	thisCwnd := cwnd()
	assert.Less(thisCwnd, lastCwnd)
}
