package ethface_test

import (
	"fmt"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/songgao/water"
	"github.com/usnistgov/ndn-dpdk/bpf"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethnetif"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"github.com/vishvananda/netlink"
	"go4.org/must"
	"inet.af/netaddr"
)

func createTUN(t testing.TB) *water.Interface {
	_, require := makeAR(t)

	cfg := water.Config{DeviceType: water.TAP}
	intf, e := water.New(cfg)
	require.NoError(e)
	t.Cleanup(func() { must.Close(intf) })

	link, e := netlink.LinkByName(intf.Name())
	require.NoError(e)
	e = netlink.LinkSetHardwareAddr(link, macaddr.MakeRandom(false))
	require.NoError(e)

	return intf
}

func TestXDPSimple(t *testing.T) {
	assert, require := makeAR(t)
	xdpProgram, e := bpf.XDP.Find("redir")
	require.NoError(e)
	tun := createTUN(t)

	port, e := ethport.New(ethport.Config{
		Config: ethnetif.Config{
			Driver:     ethnetif.DriverXDP,
			Netif:      tun.Name(),
			XDPProgram: xdpProgram,
		},
	})
	require.NoError(e)
	t.Cleanup(func() { must.Close(port) })
	localMAC := port.EthDev().HardwareAddr()
	addFace := func(loc ethport.Locator) iface.Face {
		face, e := loc.CreateFace()
		require.NoError(e)
		t.Cleanup(func() { must.Close(face) })
		return face
	}

	var locEther ethface.EtherLocator
	locEther.Local.HardwareAddr = localMAC
	locEther.Remote.Set("02:00:00:00:00:02")
	locEther.VLAN = 1987
	faceEther := addFace(locEther)

	var locUDP4 ethface.UDPLocator
	locUDP4.Local.HardwareAddr = localMAC
	locUDP4.Remote.Set("02:00:00:00:00:02")
	locUDP4.VLAN = 1987
	locUDP4.LocalIP, locUDP4.LocalUDP = netaddr.IPv4(192, 168, 2, 1), 6363
	locUDP4.RemoteIP, locUDP4.RemoteUDP = netaddr.IPv4(192, 168, 2, 2), 6363
	faceUDP4 := addFace(locUDP4)

	locUDP4p1 := locUDP4
	locUDP4p1.LocalUDP, locUDP4p1.RemoteUDP = 16363, 26363
	faceUDP4p1 := addFace(locUDP4p1)

	locUDP6 := locUDP4
	locUDP6.VLAN = 0
	locUDP6.LocalIP = netaddr.MustParseIP("fde0:fd0a:3557:a8c7:db87:639f:9bd2:0001")
	locUDP6.RemoteIP = netaddr.MustParseIP("fde0:fd0a:3557:a8c7:db87:639f:9bd2:0002")
	faceUDP6 := addFace(locUDP6)

	var locVX ethface.VxlanLocator
	locVX.EtherLocator, locVX.IPLocator = locUDP6.EtherLocator, locUDP6.IPLocator
	locVX.VXLAN = 0x887700
	locVX.InnerLocal.Set("02:00:00:00:01:01")
	locVX.InnerRemote.Set("02:00:00:00:01:02")
	faceVX := addFace(locVX)

	for i := 0; i < 500; i++ {
		interest := ndn.MakeInterest(fmt.Sprintf("/I/%d", i))
		wire, _ := tlv.EncodeFrom(interest)

		_, e = tun.Write(packetFromLayers(
			&layers.Ethernet{SrcMAC: locEther.Remote.HardwareAddr, DstMAC: locEther.Local.HardwareAddr, EthernetType: layers.EthernetTypeDot1Q},
			&layers.Dot1Q{VLANIdentifier: uint16(locEther.VLAN), Type: an.EtherTypeNDN},
			gopacket.Payload(wire),
		))
		assert.NoError(e)

		_, e = tun.Write(packetFromLayers(
			&layers.Ethernet{SrcMAC: locUDP4.Remote.HardwareAddr, DstMAC: locUDP4.Local.HardwareAddr, EthernetType: layers.EthernetTypeDot1Q},
			&layers.Dot1Q{Priority: 1, VLANIdentifier: uint16(locUDP4.VLAN), Type: layers.EthernetTypeIPv4},
			&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: locUDP4.RemoteIP.IPAddr().IP, DstIP: locUDP4.LocalIP.IPAddr().IP},
			&layers.UDP{SrcPort: layers.UDPPort(locUDP4.RemoteUDP), DstPort: layers.UDPPort(locUDP4.LocalUDP)},
			gopacket.Payload(wire),
		))
		assert.NoError(e)

		_, e = tun.Write(packetFromLayers(
			&layers.Ethernet{SrcMAC: locUDP4p1.Remote.HardwareAddr, DstMAC: locUDP4p1.Local.HardwareAddr, EthernetType: layers.EthernetTypeDot1Q},
			&layers.Dot1Q{VLANIdentifier: uint16(locUDP4p1.VLAN), Type: layers.EthernetTypeIPv4},
			&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: locUDP4p1.RemoteIP.IPAddr().IP, DstIP: locUDP4p1.LocalIP.IPAddr().IP},
			&layers.UDP{SrcPort: layers.UDPPort(locUDP4p1.RemoteUDP), DstPort: layers.UDPPort(locUDP4p1.LocalUDP)},
			gopacket.Payload(wire),
		))
		assert.NoError(e)

		_, e = tun.Write(packetFromLayers(
			&layers.Ethernet{SrcMAC: locUDP6.Remote.HardwareAddr, DstMAC: locUDP6.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv6},
			&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: locUDP6.RemoteIP.IPAddr().IP, DstIP: locUDP6.LocalIP.IPAddr().IP},
			&layers.UDP{SrcPort: layers.UDPPort(locUDP6.RemoteUDP), DstPort: layers.UDPPort(locUDP6.LocalUDP)},
			gopacket.Payload(wire),
		))
		assert.NoError(e)

		_, e = tun.Write(packetFromLayers(
			&layers.Ethernet{SrcMAC: locVX.Remote.HardwareAddr, DstMAC: locVX.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv6},
			&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: locVX.RemoteIP.IPAddr().IP, DstIP: locVX.LocalIP.IPAddr().IP},
			&layers.UDP{SrcPort: layers.UDPPort(65535 - i), DstPort: 4789},
			&layers.VXLAN{ValidIDFlag: true, VNI: uint32(locVX.VXLAN)},
			&layers.Ethernet{SrcMAC: locVX.InnerRemote.HardwareAddr, DstMAC: locVX.InnerLocal.HardwareAddr, EthernetType: an.EtherTypeNDN},
			gopacket.Payload(wire),
		))
		assert.NoError(e)
	}

	assert.EqualValues(500, faceEther.Counters().RxInterests)
	assert.EqualValues(500, faceUDP4.Counters().RxInterests)
	assert.EqualValues(500, faceUDP4p1.Counters().RxInterests)
	assert.EqualValues(500, faceUDP6.Counters().RxInterests)
	assert.EqualValues(500, faceVX.Counters().RxInterests)
}
