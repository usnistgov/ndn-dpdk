package dpdktest

import (
	"testing"
	"time"

	"ndn-dpdk/dpdk"
)

func TestTsc(t *testing.T) {
	assert, _ := makeAR(t)

	std1 := time.Now()
	tsc2 := dpdk.TscNow()
	time.Sleep(100 * time.Millisecond)
	tsc4 := dpdk.TscNow()

	assert.True(std1.Before(tsc4.ToTime()))

	tsc3 := tsc2.Add(30 * time.Millisecond)
	assert.InDelta(30*time.Millisecond, tsc3.Sub(tsc2), float64(1*time.Millisecond))

	tsc3 = tsc4.Add(-30 * time.Millisecond)
	assert.InDelta(-30*time.Millisecond, tsc3.Sub(tsc4), float64(1*time.Millisecond))
}
