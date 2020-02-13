package ethface_test

import (
	"fmt"
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
	evnCfg.NQueues = 1
	evn := dpdktestenv.NewEthVNet(evnCfg)
	defer func() {
		for _, port := range ethface.ListPorts() {
			port.Close()
		}
		evn.Close()
	}()

	var macZero dpdk.EtherAddr
	macA, _ := dpdk.ParseEtherAddr("02:00:00:00:00:01")
	macB, _ := dpdk.ParseEtherAddr("02:00:00:00:00:02")
	macC, _ := dpdk.ParseEtherAddr("02:00:00:00:00:03")

	var cfg ethface.PortConfig
	cfg.Mempools = mempools
	cfg.RxMp = mp
	cfg.RxqFrames = 64
	cfg.TxqPkts = 64
	cfg.TxqFrames = 64

	makeFace := func(dev dpdk.EthDev, local, remote dpdk.EtherAddr) *ethface.EthFace {
		loc := ethface.NewLocator(dev)
		loc.Local = local
		loc.Remote = remote
		face, e := ethface.Create(loc, cfg)
		require.NoError(e, "%s %s %s", dev.GetName(), local, remote)
		return face
	}

	faceAB := makeFace(evn.Ports[0], macA, macB)
	faceAC := makeFace(evn.Ports[0], macA, macC)
	faceAm := makeFace(evn.Ports[0], macZero, macZero)
	faceBm := makeFace(evn.Ports[1], macB, macZero)
	faceBA := makeFace(evn.Ports[1], macZero, macA)
	faceCA := makeFace(evn.Ports[2], macC, macA)

	locAm := faceAm.GetLocator().(ethface.Locator)
	assert.Equal("ether", locAm.Scheme)
	assert.Equal(evn.Ports[0].GetName(), locAm.Port)
	assert.True(locAm.Local.Equal(macA))
	assert.True(locAm.Remote.Equal(ndn.NDN_ETHER_MCAST_ADDR))

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
