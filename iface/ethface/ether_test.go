package ethface_test

import (
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethringdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/iface/ifacetestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"go4.org/must"
)

type topo3 struct {
	*ifacetestenv.Fixture
	vnet                                           *ethringdev.VNet
	macA, macB, macC                               net.HardwareAddr
	faceAB, faceAC, faceAm, faceBm, faceBA, faceCA iface.Face
}

func makeTopo3(t *testing.T, forceLinearize bool) (topo topo3) {
	_, require := makeAR(t)
	vnet := createVNet(t, ethringdev.VNetConfig{NNodes: 3})
	topo.vnet = vnet
	topo.Fixture = ifacetestenv.NewFixture(t)
	ensurePorts(t, vnet.Ports, ethport.Config{})

	topo.macA = vnet.Ports[0].HardwareAddr()
	topo.macB, _ = net.ParseMAC("02:00:00:00:00:02")
	topo.macC, _ = net.ParseMAC("02:00:00:00:00:03")

	makeFace := func(dev ethdev.EthDev, local, remote net.HardwareAddr) iface.Face {
		loc := makeEtherLocator(dev)
		if local != nil {
			loc.EthDev = dev
			loc.Local.HardwareAddr = local
		}
		loc.Remote.HardwareAddr = remote
		loc.DisableTxMultiSegOffload = forceLinearize
		face, e := loc.CreateFace()
		require.NoError(e, "%s %s %s", dev.Name(), local, remote)
		return face
	}

	topo.faceAB = makeFace(vnet.Ports[0], nil, topo.macB)
	topo.faceAC = makeFace(vnet.Ports[0], nil, topo.macC)
	topo.faceAm = makeFace(vnet.Ports[0], nil, packettransport.MulticastAddressNDN)
	topo.faceBm = makeFace(vnet.Ports[1], topo.macB, packettransport.MulticastAddressNDN)
	topo.faceBA = makeFace(vnet.Ports[1], topo.macB, topo.macA)
	topo.faceCA = makeFace(vnet.Ports[2], topo.macC, topo.macA)

	time.Sleep(time.Second)
	return topo
}

func TestTopoBA(t *testing.T) {
	topo := makeTopo3(t, false)
	topo.RunTest(topo.faceBA, topo.faceAB)
	topo.CheckCounters()
}

func TestTopoCA(t *testing.T) {
	topo := makeTopo3(t, true)
	topo.RunTest(topo.faceCA, topo.faceAC)
	topo.CheckCounters()
}

func TestTopoAm(t *testing.T) {
	assert, _ := makeAR(t)
	topo := makeTopo3(t, false)

	locAm := topo.faceAm.Locator().(ethface.EtherLocator)
	assert.Equal("ether", locAm.Scheme())
	assert.Equal(topo.macA, locAm.Local.HardwareAddr)
	assert.Equal(packettransport.MulticastAddressNDN, locAm.Remote.HardwareAddr)

	topo.RunTest(topo.faceAm, topo.faceBm)
	topo.CheckCounters()
}

func testFragmentation(t *testing.T, forceLinearize bool) {
	assert, require := makeAR(t)

	vnet := createVNet(t, ethringdev.VNetConfig{
		NNodes:          2,
		LossProbability: 0.01,
		Shuffle:         true,
	})

	fixture := ifacetestenv.NewFixture(t)
	fixture.PayloadLen = 6000
	fixture.DataFrames = 2

	ensurePorts(t, vnet.Ports, ethport.Config{MTU: 5000})

	locA := makeEtherLocator(vnet.Ports[0])
	locA.DisableTxMultiSegOffload = forceLinearize
	faceA, e := locA.CreateFace()
	require.NoError(e)

	locB := makeEtherLocator(vnet.Ports[1])
	faceB, e := locB.CreateFace()
	require.NoError(e)

	fixture.RunTest(faceA, faceB)
	fixture.CheckCounters()

	cntB := faceB.Counters()
	assert.Greater(cntB.RxReassDrops, uint64(0))
}

func TestFragmentationLinear(t *testing.T) {
	testFragmentation(t, true)
}

func TestFragmentationChained(t *testing.T) {
	testFragmentation(t, false)
}

func TestReassembly(t *testing.T) {
	assert, require := makeAR(t)
	payload := make([]byte, 6000)
	rand.Read(payload)

	vnet := createVNet(t, ethringdev.VNetConfig{NNodes: 2})
	ifacetestenv.NewFixture(t) // provides RxLoop + TxLoop
	ensurePorts(t, vnet.Ports[1:], ethport.Config{})

	portA := vnet.Ports[0]
	cfgA := ethdev.Config{}
	cfgA.AddTxQueues(1, ethdev.TxQueueConfig{})
	portA.Start(cfgA)
	locA := makeEtherLocator(vnet.Ports[0])
	txHdrA := ethport.NewTxHdr(locA, false)
	txqA := portA.TxQueues()[0]
	sendA := func(pkt *ndn.Packet) {
		b, e := tlv.EncodeFrom(pkt)
		require.NoError(e)
		m := mbuftestenv.MakePacket(b)
		txHdrA.Prepend(m, false)
		n := txqA.TxBurst(pktmbuf.Vector{m})
		assert.Equal(1, n)
	}

	locB := makeEtherLocator(vnet.Ports[1])
	locB.ReassemblerCapacity = 16
	faceB, e := locB.CreateFace()
	require.NoError(e)
	defer must.Close(faceB)
	prevCntB := faceB.Counters()
	readCntB := func() (diff iface.Counters) {
		cntB := faceB.Counters()
		diff = cntB.Since(prevCntB)
		prevCntB = cntB
		return diff
	}

	{ // reassemble 2 fragments
		fragmenter := ndn.NewLpFragmenter(5000)
		data := ndn.MakeData("/D", payload)
		frags, e := fragmenter.Fragment(data.ToPacket())
		require.NoError(e)
		require.Len(frags, 2)
		sendA(frags[0])
		sendA(frags[1])
		time.Sleep(5 * time.Millisecond)
		cntB := readCntB()
		assert.Equal(2, int(cntB.RxFrames))
		assert.Equal(1, int(cntB.RxReassPackets))
		assert.Equal(0, int(cntB.RxReassDrops))
		assert.Equal(1, int(cntB.RxData))
	}

	{ // reassemble 3 fragments, with reordering and duplicate
		fragmenter := ndn.NewLpFragmenter(2900)
		data := ndn.MakeData("/D", payload)
		frags, e := fragmenter.Fragment(data.ToPacket())
		require.NoError(e)
		require.Len(frags, 3)
		sendA(frags[0])
		sendA(frags[2])
		sendA(frags[2])
		sendA(frags[1])
		time.Sleep(5 * time.Millisecond)
		cntB := readCntB()
		assert.Equal(4, int(cntB.RxFrames))
		assert.Equal(1, int(cntB.RxReassPackets))
		assert.Equal(1, int(cntB.RxReassDrops))
		assert.Equal(1, int(cntB.RxData))
	}

	{ // discard packet due to unexpected FragCount change
		fragmenter := ndn.NewLpFragmenter(2900)
		data := ndn.MakeData("/D", payload)
		frags, e := fragmenter.Fragment(data.ToPacket())
		require.NoError(e)
		require.Len(frags, 3)
		frags[1].Fragment.FragCount--
		sendA(frags[0])
		sendA(frags[2])
		sendA(frags[1])
		time.Sleep(5 * time.Millisecond)
		cntB := readCntB()
		assert.Equal(3, int(cntB.RxFrames))
		assert.Equal(0, int(cntB.RxReassPackets))
		assert.Equal(3, int(cntB.RxReassDrops))
		assert.Equal(0, int(cntB.RxData))
	}

	{ // too many incomplete packets
		fragmenter := ndn.NewLpFragmenter(4000)
		secondFrag := make([]*ndn.Packet, 200)
		for i := range secondFrag {
			data := ndn.MakeData(fmt.Sprintf("/D/%d", i), payload)
			frags, e := fragmenter.Fragment(data.ToPacket())
			require.NoError(e)
			require.Len(frags, 2)
			sendA(frags[0])
			secondFrag[i] = frags[1]
			switch {
			case i == 50:
				sendA(secondFrag[40]) // within reassembler capacity, can reassemble
				sendA(secondFrag[20]) // exceed reassembler capacity, cannot reassemble
				fallthrough
			case i >= 100:
				sendA(frags[1])
			}
			time.Sleep(time.Millisecond)
		}
		time.Sleep(5 * time.Millisecond)
		cntB := readCntB()
		assert.LessOrEqual(int(cntB.RxFrames), 303)
		assert.GreaterOrEqual(int(cntB.RxFrames), 303-locB.ReassemblerCapacity)
		assert.Equal(102, int(cntB.RxReassPackets))
		assert.LessOrEqual(int(cntB.RxReassDrops), 99)
		assert.GreaterOrEqual(int(cntB.RxReassDrops), 99-locB.ReassemblerCapacity)
		assert.Equal(102, int(cntB.RxData))
		// incomplete packets are left in the reassembler; do not add another test after this
	}
}
