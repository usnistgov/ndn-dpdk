package ethface_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ifacetestenv"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

type ethTestTopology struct {
	*ifacetestenv.Fixture
	vnet                                           *ethdev.VNet
	macZero, macA, macB, macC                      ethdev.EtherAddr
	faceAB, faceAC, faceAm, faceBm, faceBA, faceCA iface.Face
}

func makeTopo(t *testing.T) (topo ethTestTopology) {
	_, require := makeAR(t)
	topo.Fixture = ifacetestenv.NewFixture(t)

	var vnetCfg ethdev.VNetConfig
	vnetCfg.RxPool = ndnitestenv.Packet.Pool()
	vnetCfg.NNodes = 3
	vnet := ethdev.NewVNet(vnetCfg)
	topo.vnet = vnet

	topo.macA, _ = ethdev.ParseEtherAddr("02:00:00:00:00:01")
	topo.macB, _ = ethdev.ParseEtherAddr("02:00:00:00:00:02")
	topo.macC, _ = ethdev.ParseEtherAddr("02:00:00:00:00:03")

	makeFace := func(dev ethdev.EthDev, local, remote ethdev.EtherAddr) iface.Face {
		loc := ethface.NewLocator(dev)
		loc.Local = local
		loc.Remote = remote
		face, e := ethface.Create(loc, ethPortCfg)
		require.NoError(e, "%s %s %s", dev.Name(), local, remote)
		return face
	}

	topo.faceAB = makeFace(vnet.Ports[0], topo.macA, topo.macB)
	topo.faceAC = makeFace(vnet.Ports[0], topo.macA, topo.macC)
	topo.faceAm = makeFace(vnet.Ports[0], topo.macZero, topo.macZero)
	topo.faceBm = makeFace(vnet.Ports[1], topo.macB, topo.macZero)
	topo.faceBA = makeFace(vnet.Ports[1], topo.macZero, topo.macA)
	topo.faceCA = makeFace(vnet.Ports[2], topo.macC, topo.macA)

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

	topo.RunTest(topo.faceBA, topo.faceAB)
	topo.CheckCounters()
}

func TestEthFaceCA(t *testing.T) {
	topo := makeTopo(t)
	defer topo.Close()

	topo.RunTest(topo.faceCA, topo.faceAC)
	topo.CheckCounters()
}

func TestEthFaceAm(t *testing.T) {
	assert, _ := makeAR(t)
	topo := makeTopo(t)
	defer topo.Close()

	locAm := topo.faceAm.Locator().(ethface.Locator)
	assert.Equal("ether", locAm.Scheme)
	assert.Equal(topo.vnet.Ports[0].Name(), locAm.Port)
	assert.True(locAm.Local.Equal(topo.macA))
	assert.True(locAm.Remote.Equal(ethface.NdnMcastAddr))

	topo.RunTest(topo.faceAm, topo.faceBm)
	topo.CheckCounters()
}

func TestFragmentation(t *testing.T) {
	assert, require := makeAR(t)
	fixture := ifacetestenv.NewFixture(t)
	defer fixture.Close()
	fixture.PayloadLen = 6000
	fixture.DataFrames = 2

	var vnetCfg ethdev.VNetConfig
	vnetCfg.RxPool = ndnitestenv.Packet.Pool()
	vnetCfg.NNodes = 2
	vnetCfg.LossProbability = 0.01
	vnetCfg.Shuffle = true
	vnet := ethdev.NewVNet(vnetCfg)
	ealthread.Launch(vnet)
	time.Sleep(time.Second)

	portCfg := ethPortCfg
	portCfg.Mtu = 5000
	portCfg.SkipSetMtu = true
	faceA, e := ethface.Create(ethface.NewLocator(vnet.Ports[0]), portCfg)
	require.NoError(e)
	faceB, e := ethface.Create(ethface.NewLocator(vnet.Ports[1]), portCfg)
	require.NoError(e)

	fixture.RunTest(faceA, faceB)
	fixture.CheckCounters()

	cntB := faceB.ReadCounters()
	assert.Greater(cntB.ReassDrops, uint64(0))
}
