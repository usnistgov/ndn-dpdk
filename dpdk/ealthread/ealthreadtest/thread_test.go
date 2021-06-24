package ealthreadtest

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

func TestThread(t *testing.T) {
	defer ealthread.AllocClear()
	assert, require := makeAR(t)

	th := newTestThread()
	defer th.Close()
	assert.False(th.LCore().Valid())
	require.NoError(ealthread.AllocThread(th))
	assert.True(th.LCore().Valid())
	assert.False(th.IsRunning())
	loadStat0 := th.ThreadLoadStat()

	ealthread.Launch(th)
	assert.True(th.IsRunning())
	time.Sleep(50 * time.Millisecond)

	require.NoError(th.Stop())
	assert.False(th.IsRunning())
	assert.Greater(th.GetN(), 0)
	loadStat1 := th.ThreadLoadStat()

	loadStat := loadStat1.Sub(loadStat0)
	assert.InEpsilon(loadStat.EmptyPolls, 0.2*float64(th.GetN()), 0.01)
	assert.InEpsilon(loadStat.ValidPolls, 0.8*float64(th.GetN()), 0.01)
	assert.InEpsilon(loadStat.ItemsPerPoll, 2.5, 0.01)
}
