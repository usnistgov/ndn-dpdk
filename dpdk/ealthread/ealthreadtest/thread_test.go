package ealthreadtest

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

func TestThread(t *testing.T) {
	assert, require := makeAR(t)
	defer ealthread.DefaultAllocator.Clear()

	th := newTestThread()
	defer th.Close()
	assert.False(th.LCore().Valid())
	require.NoError(ealthread.AllocThread(th))
	assert.True(th.LCore().Valid())
	assert.False(th.IsRunning())

	ealthread.Launch(th)
	assert.True(th.IsRunning())
	time.Sleep(5 * time.Millisecond)

	require.NoError(th.Stop())
	assert.False(th.IsRunning())
	assert.Greater(th.GetN(), 0)
}
