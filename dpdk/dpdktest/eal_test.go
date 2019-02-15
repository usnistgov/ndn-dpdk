package dpdktest

import (
	"testing"

	"ndn-dpdk/dpdk"
)

func TestEal(t *testing.T) {
	assert, require := makeAR(t)

	assert.Equal([]string{"testprog", "X"}, initEalRemainingArgs)

	master := dpdk.GetMasterLCore()
	assert.Equal(dpdk.LCore(0), master)
	assert.True(master.IsValid())
	assert.True(master.IsMaster())

	slaves := dpdk.ListSlaveLCores()
	require.Equal([]dpdk.LCore{2, 3}, slaves)
	for _, slave := range slaves {
		assert.True(slave.IsValid())
		assert.False(slave.IsMaster())
		assert.Equal(dpdk.LCORE_STATE_WAIT, slave.GetState())
	}

	isSlaveExecuted := false
	slaves[0].RemoteLaunch(func() int {
		assert.Equal(dpdk.LCore(2), dpdk.GetCurrentLCore())
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
