package ethface_test

import (
	"os"
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
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go4.org/must"
)

func TestMain(m *testing.M) {
	if len(os.Args) >= 2 && os.Args[1] == memifbridgeArg {
		memifbridgeHelper()
		os.Exit(0)
	}

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
// ifacetestenv.NewFixture should be called after this, so that fixture cleanup (including
// closing faces) occurs before VNet is closed.
func createVNet(t *testing.T, cfg ethringdev.VNetConfig) *ethringdev.VNet {
	_, require := makeAR(t)
	cfg.RxPool = ndni.PacketMempool.Get(eal.NumaSocket{})
	vnet, e := ethringdev.NewVNet(cfg)
	require.NoError(e)
	t.Cleanup(func() { must.Close(vnet) })
	ealthread.AllocLaunch(vnet)
	return vnet
}

// ensurePorts creates a Port for each EthDev on the VNet, if it doesn't already have one.
//
// ifacetestenv.NewFixture should be called before this, so that RxLoop and TxLoop exist.
func ensurePorts(t *testing.T, devs []ethdev.EthDev, cfg ethface.PortConfig) {
	_, require := makeAR(t)
	for _, dev := range devs {
		if ethface.FindPort(dev) != nil {
			continue
		}
		cfg.EthDev = dev
		_, e := ethface.NewPort(cfg)
		require.NoError(e)
	}
}

func makeEtherLocator(dev ethdev.EthDev) (loc ethface.EtherLocator) {
	loc.Local.HardwareAddr = dev.HardwareAddr()
	loc.Remote.HardwareAddr = packettransport.MulticastAddressNDN
	return
}

func parseLocator(j string) iface.Locator {
	var locw iface.LocatorWrapper
	fromJSON(j, &locw)
	return locw.Locator
}

func packetFromLayers(hdrs ...gopacket.SerializableLayer) *pktmbuf.Packet {
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

	buf := gopacket.NewSerializeBuffer()
	e := gopacket.SerializeLayers(buf, gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}, hdrs...)
	if e != nil {
		panic(e)
	}
	return makePacket(mbuftestenv.Headroom(0), buf.Bytes())
}
