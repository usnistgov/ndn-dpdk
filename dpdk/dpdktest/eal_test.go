package dpdktest

import (
	"testing"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestEal(t *testing.T) {
	assert, require := makeAR(t)
	eal := dpdktestenv.Eal

	assert.Equal([]string{"testprog", "X"}, eal.Args)

	assert.Equal(dpdk.LCore(0), eal.Master)
	assert.True(eal.Master.IsValid())
	assert.True(eal.Master.IsMaster())
	require.Equal([]dpdk.LCore{2, 3}, eal.Slaves)
	for _, slave := range eal.Slaves {
		assert.True(slave.IsValid())
		assert.False(slave.IsMaster())
		assert.Equal(dpdk.LCORE_STATE_WAIT, slave.GetState())
	}

	isSlaveExecuted := false
	eal.Slaves[0].RemoteLaunch(func() int {
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
	assert.Equal(66, eal.Slaves[0].Wait())
	assert.True(isSlaveExecuted)
	assert.Equal(0, eal.Slaves[0].Wait())
}
