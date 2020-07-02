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

	require.Len(eal.Workers, 5)
	workersSet := make(map[eal.LCore]bool)
	for _, worker := range eal.Workers {
		workersSet[worker] = true
		assert.True(worker.Valid())
		assert.False(worker.IsBusy())
	}
	require.Len(workersSet, 5)

	isWorkerExecuted := false
	eal.Workers[0].RemoteLaunch(cptr.IntFunction(func() int {
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
