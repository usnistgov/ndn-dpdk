package ndnlayer_test

import (
	"net"
	"net/netip"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndnlayer"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"golang.org/x/exp/slices"
)

var (
	makeAR       = testenv.MakeAR
	bytesFromHex = testenv.BytesFromHex
	bytesEqual   = testenv.BytesEqual
	nameEqual    = ndntestenv.NameEqual
)

var (
	sampleWire = bytesFromHex("641D pittoken=6203B0B1B2 payload=5016 " +
		"data=(0614 name=0703080141 content=1506C0C1C2C3C4C5 siginfo=16031B01C8 sigvalue=1700)")
	samplePayload  = bytesFromHex("C0C1C2C3C4C5")
	replacePayload = bytesFromHex("F0F1F2F3F4F5")
)

func serializePacket(isUDP bool, l ...gopacket.SerializableLayer) []byte {
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x01},
		DstMAC:       net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x02},
		EthernetType: ndnlayer.EthernetTypeNDN,
	}

	if isUDP {
		eth.EthernetType = layers.EthernetTypeIPv6
		ip6 := &layers.IPv6{
			Version:    6,
			NextHeader: layers.IPProtocolUDP,
			HopLimit:   64,
			SrcIP:      netip.MustParseAddr("fde0:fd0a:3557:a8c7:db87:639f:9bd2:0001").AsSlice(),
			DstIP:      netip.MustParseAddr("fde0:fd0a:3557:a8c7:db87:639f:9bd2:0002").AsSlice(),
		}
		udp := &layers.UDP{
			SrcPort: 16363,
			DstPort: ndnlayer.UDPPortNDN,
		}
		udp.SetNetworkLayerForChecksum(ip6)
		l = slices.Insert[[]gopacket.SerializableLayer, gopacket.SerializableLayer](l, 0, eth, ip6, udp)
	} else {
		l = slices.Insert[[]gopacket.SerializableLayer, gopacket.SerializableLayer](l, 0, eth)
	}

	w := gopacket.NewSerializeBuffer()
	gopacket.SerializeLayers(w, gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}, l...)
	return w.Bytes()
}

func checkNDNLayers(t testing.TB, tlvL *ndnlayer.TLV, ndnL *ndnlayer.NDN) {
	assert, require := makeAR(t)

	require.NotNil(tlvL)
	bytesEqual(assert, sampleWire, tlvL.LayerContents())
	bytesEqual(assert, sampleWire, tlvL.LayerPayload())

	require.NotNil(ndnL)
	bytesEqual(assert, sampleWire, ndnL.LayerContents())
	bytesEqual(assert, samplePayload, ndnL.LayerPayload())
	bytesEqual(assert, samplePayload, ndnL.Payload())
	if assert.NotNil(ndnL.Packet.Data) {
		nameEqual(assert, "/A", ndnL.Packet.Data)
	}
}

func checkZeroCopy(t testing.TB, ndnL *ndnlayer.NDN, wire []byte) {
	assert, _ := makeAR(t)
	assert.NotContains(string(wire), string(replacePayload))
	assert.Equal(len(replacePayload), copy(ndnL.Payload(), replacePayload))
	assert.Contains(string(wire), string(replacePayload))
}

func TestDecodePacket(t *testing.T) {
	_, require := makeAR(t)
	wire := serializePacket(false, gopacket.Payload(sampleWire))

	pkt := gopacket.NewPacket(wire, layers.LayerTypeEthernet, gopacket.NoCopy)
	if errL := pkt.ErrorLayer(); errL != nil {
		require.NoError(errL.Error(), gopacket.LayerDump(errL))
	}

	tlvL, _ := pkt.Layer(ndnlayer.LayerTypeTLV).(*ndnlayer.TLV)
	ndnL, _ := pkt.ApplicationLayer().(*ndnlayer.NDN)
	checkNDNLayers(t, tlvL, ndnL)
	checkZeroCopy(t, ndnL, wire)
}

func TestDecodeLayers(t *testing.T) {
	_, require := makeAR(t)
	wire := serializePacket(true, gopacket.Payload(sampleWire))

	var (
		eth     layers.Ethernet
		ip4     layers.IPv4
		ip6     layers.IPv6
		udp     layers.UDP
		tlvL    ndnlayer.TLV
		ndnL    ndnlayer.NDN
		payload gopacket.Payload
	)
	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &eth, &ip4, &ip6, &udp, &tlvL, &ndnL, &payload)

	var decoded []gopacket.LayerType
	require.NoError(parser.DecodeLayers(wire, &decoded))
	require.Equal([]gopacket.LayerType{
		layers.LayerTypeEthernet,
		layers.LayerTypeIPv6,
		layers.LayerTypeUDP,
		ndnlayer.LayerTypeTLV,
		ndnlayer.LayerTypeNDN,
		gopacket.LayerTypePayload,
	}, decoded)

	checkNDNLayers(t, &tlvL, &ndnL)
	checkZeroCopy(t, &ndnL, wire)
}

func TestEncodePacket(t *testing.T) {
	assert, require := makeAR(t)

	interest := ndn.MakeInterest("/I")
	wire := serializePacket(true, &ndnlayer.NDN{Packet: interest.ToPacket()})

	pkt := gopacket.NewPacket(wire, layers.LayerTypeEthernet, gopacket.Lazy)
	if errL := pkt.ErrorLayer(); errL != nil {
		require.NoError(errL.Error(), gopacket.LayerDump(errL))
	}

	ndnL, ok := pkt.Layer(ndnlayer.LayerTypeNDN).(*ndnlayer.NDN)
	require.True(ok)
	if assert.NotNil(ndnL.Packet.Interest) {
		nameEqual(assert, interest, ndnL.Packet.Interest)
	}
}

func TestEncodeField(t *testing.T) {
	assert, require := makeAR(t)

	field := tlv.TLVNNI(0x20, 1)
	wire := serializePacket(false, ndnlayer.SerializeFrom(field))

	pkt := gopacket.NewPacket(wire, layers.LayerTypeEthernet, gopacket.Default)
	assert.NotNil(pkt.ErrorLayer())

	tlvL, ok := pkt.Layer(ndnlayer.LayerTypeTLV).(*ndnlayer.TLV)
	require.True(ok)
	assert.EqualValues(0x20, tlvL.Element.Type)
	assert.Equal(1, tlvL.Element.Length())
}
