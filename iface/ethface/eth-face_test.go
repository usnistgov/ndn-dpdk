package ethface_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/ifacetestfixture"
	"ndn-dpdk/ndn"
)

func TestEthFace(t *testing.T) {
	assert, require := dpdktestenv.MakeAR(t)

	mp, mempools := ifacetestfixture.MakeMempools()

	var evnCfg dpdktestenv.EthVNetConfig
	evnCfg.NNodes = 3
	evnCfg.NQueues = 2
	evn := dpdktestenv.NewEthVNet(evnCfg)
	defer evn.Close()

	macA, _ := net.ParseMAC("02:00:00:00:00:01")
	macB, _ := net.ParseMAC("02:00:00:00:00:02")
	macC, _ := net.ParseMAC("02:00:00:00:00:03")

	var cfgA ethface.PortConfig
	cfgA.Mempools = mempools
	cfgA.EthDev = evn.Ports[0]
	cfgA.RxMp = mp
	cfgA.NRxThreads = evnCfg.NQueues
	cfgA.RxqFrames = 64
	cfgA.TxqPkts = 64
	cfgA.TxqFrames = 64
	cfgA.Local = macA
	cfgA.Multicast = true
	cfgA.Unicast = []net.HardwareAddr{macB, macC}
	portA, e := ethface.NewPort(cfgA)
	require.NoError(e)
	defer portA.Close()

	cfgB := cfgA
	cfgB.EthDev = evn.Ports[1]
	cfgB.Local = macB
	cfgB.Unicast = []net.HardwareAddr{macA}
	portB, e := ethface.NewPort(cfgB)
	require.NoError(e)
	defer portB.Close()

	cfgC := cfgB
	cfgC.EthDev = evn.Ports[2]
	cfgC.Local = macC
	cfgC.Multicast = false
	portC, e := ethface.NewPort(cfgC)
	require.NoError(e)
	defer portC.Close()

	faceAB := portA.ListUnicastFaces()[0]
	faceAC := portA.ListUnicastFaces()[1]
	faceAm := portA.GetMulticastFace()
	faceBA := portB.ListUnicastFaces()[0]
	faceBm := portB.GetMulticastFace()
	faceCA := portC.ListUnicastFaces()[0]

	locAm := faceAm.GetLocator().(ethface.Locator)
	assert.Equal("ether", locAm.Scheme)
	assert.Equal(portA.GetEthDev().GetName(), locAm.Port)
	assert.Equal(macA, locAm.Local)
	assert.Equal(ndn.GetEtherMcastAddr(), locAm.Remote)

	evn.LaunchBridge(dpdk.ListSlaveLCores()[3])
	time.Sleep(time.Second)

	fixtureBA := ifacetestfixture.New(t, faceAB, faceBA)
	fixtureBA.AddRxDiscard(faceCA)
	fixtureBA.RunTest()
	fixtureBA.CheckCounters()

	fixtureCA := ifacetestfixture.New(t, faceAC, faceCA)
	fixtureCA.AddRxDiscard(faceBm)
	fixtureCA.RunTest()
	fixtureCA.CheckCounters()

	fixtureAm := ifacetestfixture.New(t, faceAm, faceBm)
	fixtureAm.AddRxDiscard(faceCA)
	fixtureAm.RunTest()
	fixtureAm.CheckCounters()

	fmt.Println("evn.NDrops", evn.NDrops)
	fmt.Println("portA", evn.Ports[0].GetStats())
	fmt.Println("portB", evn.Ports[1].GetStats())
	fmt.Println("portC", evn.Ports[2].GetStats())
	fmt.Println("faceAB", faceAB.ReadCounters())
	fmt.Println("faceAC", faceAC.ReadCounters())
	fmt.Println("faceAm", faceAm.ReadCounters())
	fmt.Println("faceBA", faceBA.ReadCounters())
	fmt.Println("faceBm", faceBm.ReadCounters())
	fmt.Println("faceCA", faceCA.ReadCounters())
}
