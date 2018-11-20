package dpdktest

import (
	"testing"
	"time"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestThread(t *testing.T) {
	assert, require := makeAR(t)
	eal := dpdktestenv.Eal
	require.True(len(eal.Slaves) >= 1)

	th := newTestThread()
	assert.Implements((*dpdk.IThread)(nil), th)
	th.SetLCore(eal.Slaves[0])
	assert.Equal(eal.Slaves[0], th.GetLCore())
	assert.False(th.IsRunning())

	require.NoError(th.Launch())
	assert.Equal(eal.Slaves[0], th.GetLCore())
	assert.True(th.IsRunning())
	time.Sleep(5 * time.Millisecond)

	require.NoError(th.Stop())
	assert.False(th.IsRunning())
	assert.True(th.GetN() > 0)

	require.NoError(th.Close())
}
