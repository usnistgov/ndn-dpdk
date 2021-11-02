package ethface_test

import (
	"os"
	"testing"

	"github.com/google/gopacket"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
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
