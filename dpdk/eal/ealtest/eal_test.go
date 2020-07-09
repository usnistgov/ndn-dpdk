package ealtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func TestEal(t *testing.T) {
	assert, require := makeAR(t)

	assert.Equal([]string{"testprog", "c7f36046-faa5-46dc-9855-e93d00217b8f"}, initEalRemainingArgs)

	assert.True(eal.Initial.Valid())
	assert.NotNil(eal.MainThread)
	assert.NotNil(eal.MainReadSide)
	assert.NotEmpty(eal.Sockets)

	require.Len(eal.Workers, 5)
	workersSet := make(map[eal.LCore]bool)
	for _, worker := range eal.Workers {
		workersSet[worker] = true
		assert.True(worker.Valid())
		assert.False(worker.IsBusy())
	}
	require.Len(workersSet, 5)

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
	assert.Equal(0, eal.Workers[0].Wait())
}

func TestEalJson(t *testing.T) {
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
