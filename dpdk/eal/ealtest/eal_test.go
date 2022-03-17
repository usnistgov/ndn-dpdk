package ealtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
)

func TestEal(t *testing.T) {
	assert, require := makeAR(t)

	assert.True(eal.MainLCore.Valid())
	assert.NotNil(eal.MainThread)
	assert.NotNil(eal.MainReadSide)
	assert.NotEmpty(eal.Sockets)

	require.Len(eal.Workers, ealtestenv.WantLCores-1)
	workersSet := map[eal.LCore]bool{}
	for _, worker := range eal.Workers {
		workersSet[worker] = true
		assert.True(worker.Valid())
		assert.False(worker.IsBusy())
	}
	require.Len(workersSet, ealtestenv.WantLCores-1)

	isWorkerExecuted := false
	eal.Workers[0].RemoteLaunch(cptr.Func0.Int(func() int {
		assert.Equal(eal.Workers[0], eal.CurrentLCore())
		isWorkerExecuted = true

		done := make(chan bool)
		go func() {
			assert.False(eal.CurrentLCore().Valid())
			done <- true
		}()
		<-done

		return 66
	}))
	assert.Equal(66, eal.Workers[0].Wait())
	assert.True(isWorkerExecuted)
}

func TestEalJSON(t *testing.T) {
	assert, _ := makeAR(t)

	var lc eal.LCore
	assert.Equal("null", toJSON(lc))

	lc = eal.LCoreFromID(5)
	assert.Equal("5", toJSON(lc))

	var socket eal.NumaSocket
	assert.Equal("null", toJSON(socket))

	socket = eal.NumaSocketFromID(1)
	assert.Equal("1", toJSON(socket))
}
