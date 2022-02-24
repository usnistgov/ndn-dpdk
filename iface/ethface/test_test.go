package ethface_test

import (
	"sync"
	"testing"

	"github.com/google/gopacket"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethringdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go4.org/must"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()

	pktmbuf.Direct.Update(pktmbuf.PoolConfig{
		Dataroom: 9000, // needed by fragmentation test case
		Capacity: 16383,
	})

	testenv.Exit(m.Run())
}

var (
	makeAR     = testenv.MakeAR
	fromJSON   = testenv.FromJSON
	makePacket = mbuftestenv.MakePacket
)

// createVNet creates a VNet from config template, and schedules its cleanup.
//
// Faces and ports must be closed before closing VNet. One way to ensure this condition is
// calling `defer iface.CloseAll()` or `ifacetestenv.NewFixture` after this function.
func createVNet(t testing.TB, cfg ethringdev.VNetConfig) *ethringdev.VNet {
	_, require := makeAR(t)
	cfg.RxPool = ndni.PacketMempool.Get(eal.NumaSocket{})
	vnet, e := ethringdev.NewVNet(cfg)
	require.NoError(e)
	t.Cleanup(func() { must.Close(vnet) })
	ealthread.AllocLaunch(vnet)
	return vnet
}

// ensurePorts creates a Port for each EthDev on the VNet, if it doesn't already have one.
func ensurePorts(t testing.TB, devs []ethdev.EthDev, cfg ethport.Config) {
	_, require := makeAR(t)
	for _, dev := range devs {
		if ethport.Find(dev) != nil {
			continue
		}
		cfg.EthDev = dev
		_, e := ethport.New(cfg)
		require.NoError(e)
	}
}

func makeEtherLocator(dev ethdev.EthDev) (loc ethface.EtherLocator) {
	loc.Local.HardwareAddr = dev.HardwareAddr()
	loc.Remote.HardwareAddr = packettransport.MulticastAddressNDN
	return
}

func parseLocator(j string) ethport.Locator {
	var locw iface.LocatorWrapper
	fromJSON(j, &locw)
	return locw.Locator.(ethport.Locator)
}

var serializeBufferPool = sync.Pool{
	New: func() interface{} { return gopacket.NewSerializeBuffer() },
}

func packetFromLayers(hdrs ...gopacket.SerializableLayer) []byte {
	type TransportLayer interface {
		SetNetworkLayerForChecksum(l gopacket.NetworkLayer) error
	}
	var netLayer gopacket.NetworkLayer
	for _, hdr := range hdrs {
		switch layer := hdr.(type) {
		case gopacket.NetworkLayer:
			netLayer = layer
		case TransportLayer:
			if netLayer != nil {
				layer.SetNetworkLayerForChecksum(netLayer)
			}
		}
	}

	buf := serializeBufferPool.Get().(gopacket.SerializeBuffer)
	defer func() {
		buf.Clear()
		serializeBufferPool.Put(buf)
	}()

	e := gopacket.SerializeLayers(buf, gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}, hdrs...)
	if e != nil {
		panic(e)
	}
	return buf.Bytes()
}

func pktmbufFromLayers(hdrs ...gopacket.SerializableLayer) *pktmbuf.Packet {
	buf := packetFromLayers(hdrs...)
	return makePacket(mbuftestenv.Headroom(0), buf)
}
