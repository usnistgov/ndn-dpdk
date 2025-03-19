//go:build linux

package ethertransport_test

import (
	"net"
	"os"
	"strings"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn/ethertransport"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

var (
	makeAR = testenv.MakeAR
)

func TestAddress(t *testing.T) {
	assert, _ := makeAR(t)
	assert.True(macaddr.IsMulticast(ethertransport.MulticastAddressNDN))
}

func TestVf(t *testing.T) {
	_, require := makeAR(t)

	line, ok := os.LookupEnv("ETHERTRANSPORTTEST_VF")
	if !ok {
		// ETHERTRANSPORTTEST_VF syntax: ifname0,ifname1
		t.Skip("VF test disabled; rerun test suite and specify two netifs in ETHERTRANSPORTTEST_VF environ.")
	}
	tokens := strings.Split(line, ",")
	require.Len(tokens, 2)

	ifname0, ifname1 := tokens[0], tokens[1]
	netif0, e := net.InterfaceByName(ifname0)
	require.NoError(e)
	netif1, e := net.InterfaceByName(ifname1)
	require.NoError(e)

	var cfgA ethertransport.Config
	cfgA.MTU = min(netif0.MTU, netif1.MTU)
	cfgA.Local.HardwareAddr = netif0.HardwareAddr
	cfgA.Remote.HardwareAddr = netif1.HardwareAddr
	cfgB := cfgA
	cfgB.Local, cfgB.Remote = cfgA.Remote, cfgA.Local

	trA, e := ethertransport.New(ifname0, cfgA)
	require.NoError(e)
	trB, e := ethertransport.New(ifname1, cfgB)
	require.NoError(e)

	var c ndntestenv.L3FaceTester
	c.CheckTransport(t, trA, trB)
}
