package ealtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func TestEal(t *testing.T) {
	assert, require := makeAR(t)

	assert.Equal([]string{"testprog", "c7f36046-faa5-46dc-9855-e93d00217b8f"}, initEalRemainingArgs)

	initial := eal.GetInitialLCore()
	assert.True(initial.Valid())

	workers := eal.ListWorkerLCores()
	require.Len(workers, 5)
	workersSet := make(map[eal.LCore]bool)
	for _, worker := range workers {
		workersSet[worker] = true
		assert.True(worker.Valid())
		assert.False(worker.IsBusy())
	}
	require.Len(workersSet, 5)

	isWorkerExecuted := false
	workers[0].RemoteLaunch(func() int {
		assert.Equal(workers[0], eal.GetCurrentLCore())
		isWorkerExecuted = true

		done := make(chan bool)
		go func() {
			assert.False(eal.GetCurrentLCore().Valid())
			done <- true
		}()
		<-done

		return 66
	})
	assert.Equal(66, workers[0].Wait())
	assert.True(isWorkerExecuted)
	assert.Equal(0, workers[0].Wait())
}
