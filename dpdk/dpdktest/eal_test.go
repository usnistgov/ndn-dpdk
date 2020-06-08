package dpdktest

import (
	"testing"

	"ndn-dpdk/dpdk"
)

func TestEal(t *testing.T) {
	assert, require := makeAR(t)

	assert.Equal([]string{"testprog", "c7f36046-faa5-46dc-9855-e93d00217b8f"}, initEalRemainingArgs)

	master := dpdk.GetMasterLCore()
	assert.True(master.IsValid())
	assert.True(master.IsMaster())

	slaves := dpdk.ListSlaveLCores()
	require.Len(slaves, 5)
	slavesSet := make(map[int]bool)
	for _, slave := range slaves {
		slavesSet[int(slave)] = true
		assert.True(slave.IsValid())
		assert.False(slave.IsMaster())
		assert.Equal(dpdk.LCORE_STATE_WAIT, slave.GetState())
	}
	require.Len(slavesSet, 5)

	isSlaveExecuted := false
	slaves[0].RemoteLaunch(func() int {
		assert.Equal(slaves[0], dpdk.GetCurrentLCore())
		isSlaveExecuted = true

		done := make(chan bool)
		go func() {
			assert.False(dpdk.GetCurrentLCore().IsValid())
			done <- true
		}()
		<-done

		return 66
	})
	assert.Equal(66, slaves[0].Wait())
	assert.True(isSlaveExecuted)
	assert.Equal(0, slaves[0].Wait())
}
