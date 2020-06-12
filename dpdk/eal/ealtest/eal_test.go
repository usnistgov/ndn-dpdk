package ealtest

import (
	"testing"

	"ndn-dpdk/dpdk/eal"
)

func TestEal(t *testing.T) {
	assert, require := makeAR(t)

	assert.Equal([]string{"testprog", "c7f36046-faa5-46dc-9855-e93d00217b8f"}, initEalRemainingArgs)

	master := eal.GetMasterLCore()
	assert.True(master.IsValid())

	slaves := eal.ListSlaveLCores()
	require.Len(slaves, 5)
	slavesSet := make(map[eal.LCore]bool)
	for _, slave := range slaves {
		slavesSet[slave] = true
		assert.True(slave.IsValid())
		assert.False(slave.IsBusy())
	}
	require.Len(slavesSet, 5)

	isSlaveExecuted := false
	slaves[0].RemoteLaunch(func() int {
		assert.Equal(slaves[0], eal.GetCurrentLCore())
		isSlaveExecuted = true

		done := make(chan bool)
		go func() {
			assert.False(eal.GetCurrentLCore().IsValid())
			done <- true
		}()
		<-done

		return 66
	})
	assert.Equal(66, slaves[0].Wait())
	assert.True(isSlaveExecuted)
	assert.Equal(0, slaves[0].Wait())
}
