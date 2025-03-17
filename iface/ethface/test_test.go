package ethface_test

import (
	"bytes"
	"net"
	"net/netip"
	"sync"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/afpacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/stretchr/testify/require"
	"github.com/usnistgov/ndn-dpdk/bpf"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethnetif"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethringdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
	"go4.org/must"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()

	if xdpProgram, e := bpf.XDP.Find("redir"); e == nil {
		ethnetif.XDPProgram = xdpProgram
	} else {
		panic(e)
	}

	pktmbuf.Direct.Update(pktmbuf.PoolConfig{
		Dataroom: 9000, // needed by fragmentation test case
		Capacity: 16383,
	})

	testenv.Exit(m.Run())
}

var (
	makeAR       = testenv.MakeAR
	makePacket   = mbuftestenv.MakePacket
	makeInterest = ndnitestenv.MakeInterest
	randBytes    = testenv.RandBytes
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
	locw := testenv.FromJSON[iface.LocatorWrapper](j)
	return locw.Locator.(ethport.Locator)
}

var serializeBufferPool = sync.Pool{
	New: func() any { return gopacket.NewSerializeBuffer() },
}

type checksumTransportLayer interface {
	SetNetworkLayerForChecksum(l gopacket.NetworkLayer) error
}

func packetFromLayers(hdrs ...gopacket.SerializableLayer) (b []byte, discard func()) {
	var netLayer gopacket.NetworkLayer
	for _, l := range hdrs {
		switch l := l.(type) {
		case gopacket.NetworkLayer:
			netLayer = l
		case checksumTransportLayer:
			l.SetNetworkLayerForChecksum(netLayer) // ignore error when netLayer==nil
		}
	}

	buf := serializeBufferPool.Get().(gopacket.SerializeBuffer)
	if e := gopacket.SerializeLayers(buf, gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}, hdrs...); e != nil {
		panic(e)
	}
	return buf.Bytes(), func() {
		buf.Clear()
		serializeBufferPool.Put(buf)
	}
}

func pktmbufFromLayers(headroom mbuftestenv.Headroom, hdrs ...gopacket.SerializableLayer) *pktmbuf.Packet {
	b, discard := packetFromLayers(hdrs...)
	defer discard()
	return makePacket(headroom, b)
}

func dumpPcap(netif *ethnetif.NetIntf) <-chan []byte {
	tp, e := afpacket.NewTPacket(afpacket.OptInterface(netif.Name))
	if e != nil {
		panic(e)
	}

	ch := make(chan []byte)
	go func() {
		for {
			wire, _, e := tp.ReadPacketData()
			if e != nil {
				close(ch)
				tp.Close()
				tp = nil
				return
			}
			if len(wire) >= 14 && bytes.Equal(wire[6:12], netif.HardwareAddr) {
				// discard loopback packets whose source address is self
				continue
			}
			ch <- wire
		}
	}()
	return ch
}

// makeARP constructs Ethernet and ARP layers.
//
//	dstMAC: if nil, make ARP request; otherwise, make ARP reply.
func makeARP(srcMAC net.HardwareAddr, srcIP netip.Addr, dstMAC net.HardwareAddr, dstIP netip.Addr) []gopacket.SerializableLayer {
	op, ethDst := uint16(layers.ARPReply), dstMAC
	if len(dstMAC) == 0 {
		op = layers.ARPRequest
		ethDst = net.HardwareAddr{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
		dstMAC = net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	}
	return []gopacket.SerializableLayer{
		&layers.Ethernet{
			SrcMAC:       srcMAC,
			DstMAC:       ethDst,
			EthernetType: layers.EthernetTypeARP,
		},
		&layers.ARP{
			AddrType:          layers.LinkTypeEthernet,
			Protocol:          layers.EthernetTypeIPv4,
			Operation:         op,
			SourceHwAddress:   srcMAC,
			SourceProtAddress: srcIP.AsSlice(),
			DstHwAddress:      dstMAC,
			DstProtAddress:    dstIP.AsSlice(),
		},
	}
}

// makeGTPv1U constructs a GTPv1U layer.
//
//	pduType: 0=downlink, 1=uplink
func makeGTPv1U(teid uint32, pduType uint8, qfi uint8) *layers.GTPv1U {
	return &layers.GTPv1U{
		Version:      1,
		ProtocolType: 1,
		MessageType:  0xFF,
		TEID:         teid,
		GTPExtensionHeaders: []layers.GTPExtensionHeader{
			{Type: 0x85, Content: []byte{pduType << 4, qfi & 0x3F}},
		},
	}
}

func checkPacketLayers(t require.TestingT, wire []byte, expectedLayerTypes ...gopacket.LayerType) gopacket.Packet {
	assert, _ := makeAR(t)

	parsed := gopacket.NewPacket(wire, layers.LayerTypeEthernet, gopacket.NoCopy)
	actualLayerTypes := []gopacket.LayerType{}
	for _, l := range parsed.Layers() {
		actualLayerTypes = append(actualLayerTypes, l.LayerType())
	}
	assert.Equal(expectedLayerTypes, actualLayerTypes)

	var netLayer gopacket.NetworkLayer
	for _, l := range parsed.Layers() {
		switch l := l.(type) {
		case gopacket.NetworkLayer:
			netLayer = l
		case checksumTransportLayer:
			l.SetNetworkLayerForChecksum(netLayer) // ignore error when netLayer==nil
		}
	}
	e, mismatches := parsed.VerifyChecksums()
	assert.NoError(e)
	assert.Empty(mismatches)

	return parsed
}
