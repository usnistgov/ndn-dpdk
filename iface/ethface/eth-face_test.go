package ethface_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ifacetestenv"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func TestEthFace(t *testing.T) {
	assert, require := makeAR(t)
	rxPool := ndnitestenv.Packet.Pool()

	var vnetCfg ethdev.VNetConfig
	vnetCfg.RxPool = rxPool
	vnetCfg.NNodes = 3
	vnet := ethdev.NewVNet(vnetCfg)
	defer vnet.Close()

	var macZero ethdev.EtherAddr
	macA, _ := ethdev.ParseEtherAddr("02:00:00:00:00:01")
	macB, _ := ethdev.ParseEtherAddr("02:00:00:00:00:02")
	macC, _ := ethdev.ParseEtherAddr("02:00:00:00:00:03")

	var cfg ethface.PortConfig
	cfg.RxqFrames = 64
	cfg.TxqPkts = 64
	cfg.TxqFrames = 64

	makeFace := func(dev ethdev.EthDev, local, remote ethdev.EtherAddr) *ethface.EthFace {
		loc := ethface.NewLocator(dev)
		loc.Local = local
		loc.Remote = remote
		face, e := ethface.Create(loc, cfg)
		require.NoError(e, "%s %s %s", dev.Name(), local, remote)
		return face
	}

	faceAB := makeFace(vnet.Ports[0], macA, macB)
	faceAC := makeFace(vnet.Ports[0], macA, macC)
	faceAm := makeFace(vnet.Ports[0], macZero, macZero)
	faceBm := makeFace(vnet.Ports[1], macB, macZero)
	faceBA := makeFace(vnet.Ports[1], macZero, macA)
	faceCA := makeFace(vnet.Ports[2], macC, macA)

	locAm := faceAm.Locator().(ethface.Locator)
	assert.Equal("ether", locAm.Scheme)
	assert.Equal(vnet.Ports[0].Name(), locAm.Port)
	assert.True(locAm.Local.Equal(macA))
	assert.True(locAm.Remote.Equal(ethface.NdnMcastAddr))

	vnet.LaunchBridge(eal.ListWorkerLCores()[3])
	time.Sleep(time.Second)

	fixtureBA := ifacetestenv.New(t, faceAB, faceBA)
	fixtureBA.AddRxDiscard(faceCA)
	fixtureBA.RunTest()
	fixtureBA.CheckCounters()

	fixtureCA := ifacetestenv.New(t, faceAC, faceCA)
	fixtureCA.AddRxDiscard(faceBm)
	fixtureCA.RunTest()
	fixtureCA.CheckCounters()

	fixtureAm := ifacetestenv.New(t, faceAm, faceBm)
	fixtureAm.AddRxDiscard(faceCA)
	fixtureAm.RunTest()
	fixtureAm.CheckCounters()

	fmt.Println("vnet.NDrops", vnet.NDrops)
	fmt.Println("portA", vnet.Ports[0].Stats())
	fmt.Println("portB", vnet.Ports[1].Stats())
	fmt.Println("portC", vnet.Ports[2].Stats())
	fmt.Println("faceAB", faceAB.ReadCounters())
	fmt.Println("faceAC", faceAC.ReadCounters())
	fmt.Println("faceAm", faceAm.ReadCounters())
	fmt.Println("faceBA", faceBA.ReadCounters())
	fmt.Println("faceBm", faceBm.ReadCounters())
	fmt.Println("faceCA", faceCA.ReadCounters())
}
