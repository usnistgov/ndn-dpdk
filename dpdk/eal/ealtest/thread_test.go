package ealtest

import (
	"testing"
	"time"

	"ndn-dpdk/dpdk/eal"
)

func TestThread(t *testing.T) {
	assert, require := makeAR(t)
	slaves := eal.ListSlaveLCores()

	th := newTestThread()
	assert.Implements((*eal.IThread)(nil), th)
	th.SetLCore(slaves[0])
	assert.Equal(slaves[0], th.GetLCore())
	assert.False(th.IsRunning())

	require.NoError(th.Launch())
	assert.Equal(slaves[0], th.GetLCore())
	assert.True(th.IsRunning())
	time.Sleep(5 * time.Millisecond)

	require.NoError(th.Stop())
	assert.False(th.IsRunning())
	assert.True(th.GetN() > 0)

	require.NoError(th.Close())
}
