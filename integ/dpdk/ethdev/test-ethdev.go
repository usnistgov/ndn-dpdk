package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ndn-traffic-dpdk/dpdk"
	"ndn-traffic-dpdk/integ"
)

func main() {
	t := new(integ.Testing)
	defer t.Close()
	assert := assert.New(t)
	require := require.New(t)

	_, e := dpdk.NewEal([]string{"testprog", "--no-pci", "--vdev=net_ring0", "--vdev=net_ring1"})
	require.NoError(e)

	assert.EqualValues(2, dpdk.CountEthDevs())
	ethDevs := dpdk.ListEthDevs()
	assert.Equal(2, len(ethDevs))

	assert.NotEqual(ethDevs[0].GetName(), ethDevs[1].GetName())
}
