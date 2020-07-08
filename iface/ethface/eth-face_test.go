package ethface_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ifacetestenv"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

type ethTestTopology struct {
	*ifacetestenv.Fixture
	vnet                                           *ethdev.VNet
	faceAB, faceAC, faceAm, faceBm, faceBA, faceCA *ethface.EthFace
}

func makeTopo(t *testing.T) (topo ethTestTopology) {
	_, require := makeAR(t)
	topo.Fixture = ifacetestenv.New(t)
	rxPool := ndnitestenv.Packet.Pool()

	var vnetCfg ethdev.VNetConfig
	vnetCfg.RxPool = rxPool
	vnetCfg.NNodes = 3
	vnet := ethdev.NewVNet(vnetCfg)
	topo.vnet = vnet

	var macZero ethdev.EtherAddr
	macA, _ := ethdev.ParseEtherAddr("02:00:00:00:00:01")
	macB, _ := ethdev.ParseEtherAddr("02:00:00:00:00:02")
	macC, _ := ethdev.ParseEtherAddr("02:00:00:00:00:03")

	makeFace := func(dev ethdev.EthDev, local, remote ethdev.EtherAddr) *ethface.EthFace {
		loc := ethface.NewLocator(dev)
		loc.Local = local
		loc.Remote = remote
		face, e := ethface.Create(loc, ethPortCfg)
		require.NoError(e, "%s %s %s", dev.Name(), local, remote)
		return face
	}

	topo.faceAB = makeFace(vnet.Ports[0], macA, macB)
	topo.faceAC = makeFace(vnet.Ports[0], macA, macC)
	topo.faceAm = makeFace(vnet.Ports[0], macZero, macZero)
	topo.faceBm = makeFace(vnet.Ports[1], macB, macZero)
	topo.faceBA = makeFace(vnet.Ports[1], macZero, macA)
	topo.faceCA = makeFace(vnet.Ports[2], macC, macA)

	ealthread.Launch(vnet)
	time.Sleep(time.Second)
	return topo
}

func (topo *ethTestTopology) Close() error {
	topo.Fixture.Close()
	return topo.vnet.Close()
}

func TestEthFaceBA(t *testing.T) {
	topo := makeTopo(t)
	defer topo.Close()

	topo.AddRxDiscard(topo.faceCA)
	topo.RunTest(topo.faceBA, topo.faceAB)
	topo.CheckCounters()
}

func TestEthFaceCA(t *testing.T) {
	topo := makeTopo(t)
	defer topo.Close()

	topo.AddRxDiscard(topo.faceBm)
	topo.RunTest(topo.faceCA, topo.faceAC)
	topo.CheckCounters()
}

func TestEthFaceAm(t *testing.T) {
	assert, _ := makeAR(t)
	topo := makeTopo(t)
	defer topo.Close()

	macA, _ := ethdev.ParseEtherAddr("02:00:00:00:00:01")
	locAm := topo.faceAm.Locator().(ethface.Locator)
	assert.Equal("ether", locAm.Scheme)
	assert.Equal(topo.vnet.Ports[0].Name(), locAm.Port)
	assert.True(locAm.Local.Equal(macA))
	assert.True(locAm.Remote.Equal(ethface.NdnMcastAddr))

	topo.AddRxDiscard(topo.faceCA)
	topo.RunTest(topo.faceAm, topo.faceBm)
	topo.CheckCounters()
}
