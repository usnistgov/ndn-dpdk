package ethdev_test

import (
	"net"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
)

func TestTap(t *testing.T) {
	assert, require := makeAR(t)

	ifname := "ealtap0"
	local := macaddr.MakeRandomUnicast()
	dev, e := ethdev.NewTap(ifname, local)
	require.NoError(e)
	defer dev.Close()

	intf, e := net.InterfaceByName(ifname)
	require.NoError(e)
	assert.Equal(local, intf.HardwareAddr)
}
