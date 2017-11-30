package main

import (
	"ndn-traffic-dpdk/dpdk"
	"ndn-traffic-dpdk/integ"
  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/require"
)

func main() {
	t := new(integ.Testing)
	defer t.Close()

	args := []string{"testprog", "-l0,1,3", "-n1", "--no-pci", "--", "X"}
	eal, e := dpdk.NewEal(args)
	require.NoError(t, e)

	assert.Equal(t, []string{"testprog", "X"}, eal.Args)

	assert.Equal(t, dpdk.LCore(0), eal.Master)
	assert.True(t, eal.Master.IsMaster())
	require.Equal(t, []dpdk.LCore{1, 3}, eal.Slaves)
	for _, slave := range eal.Slaves {
		assert.False(t, slave.IsMaster())
		assert.Equal(t, dpdk.LCORE_STATE_WAIT, slave.GetState())
	}

	isSlaveExecuted := false
  eal.Slaves[0].RemoteLaunch(func() int {
		isSlaveExecuted = true
		return 66
	})
	assert.Equal(t, 66, eal.Slaves[0].Wait())
	assert.True(t, isSlaveExecuted)
	assert.Equal(t, 0, eal.Slaves[0].Wait())
}