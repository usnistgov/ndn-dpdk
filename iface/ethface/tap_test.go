package ethface_test

import (
	"fmt"
	"net"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

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
	"github.com/usnistgov/ndn-dpdk/ndn/ndnlayer"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/vishvananda/netlink"
	"go4.org/must"
)

func createTAP(t testing.TB) *water.Interface {
	_, require := makeAR(t)

	cfg := water.Config{DeviceType: water.TAP}
	intf, e := water.New(cfg)
	require.NoError(e)
	t.Cleanup(func() { must.Close(intf) })

	link, e := netlink.LinkByName(intf.Name())
	require.NoError(e)
	e = netlink.LinkSetHardwareAddr(link, macaddr.MakeRandomUnicast())
	require.NoError(e)

	return intf
}

func testPortTAP(t testing.TB, makeNetifConfig func(ifname string) ethnetif.Config) {
	assert, require := makeAR(t)
	tap := createTAP(t)

	port, e := ethport.New(ethport.Config{
		Config: makeNetifConfig(tap.Name()),
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
	locUDP4.EtherLocator = locEther
	locUDP4.LocalIP, locUDP4.LocalUDP = netip.MustParseAddr("192.168.2.1"), 6363
	locUDP4.RemoteIP, locUDP4.RemoteUDP = netip.MustParseAddr("192.168.2.2"), 6363
	faceUDP4 := addFace(locUDP4)

	locUDP4p1 := locUDP4
	locUDP4p1.LocalUDP, locUDP4p1.RemoteUDP = 16363, 26363
	faceUDP4p1 := addFace(locUDP4p1)

	locUDP6 := locUDP4
	locUDP6.VLAN = 0
	locUDP6.LocalIP = netip.MustParseAddr("fde0:fd0a:3557:a8c7:db87:639f:9bd2:0001")
	locUDP6.RemoteIP = netip.MustParseAddr("fde0:fd0a:3557:a8c7:db87:639f:9bd2:0002")
	faceUDP6 := addFace(locUDP6)

	var locVX ethface.VxlanLocator
	locVX.EtherLocator, locVX.IPLocator = locUDP6.EtherLocator, locUDP6.IPLocator
	locVX.VXLAN = 0x887700
	locVX.InnerLocal.Set("02:00:00:00:01:01")
	locVX.InnerRemote.Set("02:00:00:00:01:02")
	faceVX := addFace(locVX)

	var txEther, txUDP4, txUDP4p1, txUDP6, txVX, txOther atomic.Int32
	go func() {
		buf := make([]byte, port.EthDev().MTU())
		for {
			n, e := tap.Read(buf)
			if e != nil {
				break
			}

			classify, isV4 := &txOther, false
			parsed := gopacket.NewPacket(buf[:n], layers.LayerTypeEthernet, gopacket.NoCopy)
			for _, l := range parsed.Layers() {
				switch l := l.(type) {
				case *layers.Dot1Q:
					if l.Type == an.EtherTypeNDN {
						classify = &txEther
					}
				case *layers.IPv4:
					isV4 = true
				case *layers.UDP:
					if int(l.SrcPort) == locUDP4p1.LocalUDP {
						classify = &txUDP4p1
					} else if isV4 {
						classify = &txUDP4
					} else {
						classify = &txUDP6
					}
				case *layers.VXLAN:
					classify = &txVX
				}
			}
			classify.Add(1)
		}
	}()

	makeRxFrame := func(prefix string, i int) gopacket.SerializableLayer {
		interest := ndn.MakeInterest(fmt.Sprintf("/RX/%s/%d", prefix, i))
		return &ndnlayer.NDN{Packet: interest.ToPacket()}
	}
	makeTxBurst := func(prefix string, i int) []*ndni.Packet {
		return []*ndni.Packet{makeInterest(fmt.Sprintf("/TX/%s/%d", prefix, i))}
	}

	for i := 0; i < 500; i++ {
		time.Sleep(10 * time.Millisecond)

		_, e = writeToFromLayers(tap,
			&layers.Ethernet{SrcMAC: locEther.Remote.HardwareAddr, DstMAC: locEther.Local.HardwareAddr, EthernetType: layers.EthernetTypeDot1Q},
			&layers.Dot1Q{VLANIdentifier: uint16(locEther.VLAN), Type: an.EtherTypeNDN},
			makeRxFrame("Ether", i),
		)
		assert.NoError(e)
		iface.TxBurst(faceEther.ID(), makeTxBurst("Ether", i))

		_, e = writeToFromLayers(tap,
			&layers.Ethernet{SrcMAC: locUDP4.Remote.HardwareAddr, DstMAC: locUDP4.Local.HardwareAddr, EthernetType: layers.EthernetTypeDot1Q},
			&layers.Dot1Q{VLANIdentifier: uint16(locUDP4.VLAN), Type: layers.EthernetTypeIPv4},
			&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: net.IP(locUDP4.RemoteIP.AsSlice()), DstIP: net.IP(locUDP4.LocalIP.AsSlice())},
			&layers.UDP{SrcPort: layers.UDPPort(locUDP4.RemoteUDP), DstPort: layers.UDPPort(locUDP4.LocalUDP)},
			makeRxFrame("UDP4", i),
		)
		assert.NoError(e)
		iface.TxBurst(faceUDP4.ID(), makeTxBurst("UDP4", i))

		_, e = writeToFromLayers(tap,
			&layers.Ethernet{SrcMAC: locUDP4p1.Remote.HardwareAddr, DstMAC: locUDP4p1.Local.HardwareAddr, EthernetType: layers.EthernetTypeDot1Q},
			&layers.Dot1Q{Priority: 1, VLANIdentifier: uint16(locUDP4p1.VLAN), Type: layers.EthernetTypeIPv4},
			&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: net.IP(locUDP4p1.RemoteIP.AsSlice()), DstIP: net.IP(locUDP4p1.LocalIP.AsSlice())},
			&layers.UDP{SrcPort: layers.UDPPort(locUDP4p1.RemoteUDP), DstPort: layers.UDPPort(locUDP4p1.LocalUDP)},
			makeRxFrame("UDP4p1", i),
		)
		assert.NoError(e)
		iface.TxBurst(faceUDP4p1.ID(), makeTxBurst("UDP4p1", i))

		_, e = writeToFromLayers(tap,
			&layers.Ethernet{SrcMAC: locUDP6.Remote.HardwareAddr, DstMAC: locUDP6.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv6},
			&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: net.IP(locUDP6.RemoteIP.AsSlice()), DstIP: net.IP(locUDP6.LocalIP.AsSlice())},
			&layers.UDP{SrcPort: layers.UDPPort(locUDP6.RemoteUDP), DstPort: layers.UDPPort(locUDP6.LocalUDP)},
			makeRxFrame("UDP6", i),
		)
		assert.NoError(e)
		iface.TxBurst(faceUDP6.ID(), makeTxBurst("UDP6", i))

		_, e = writeToFromLayers(tap,
			&layers.Ethernet{SrcMAC: locVX.Remote.HardwareAddr, DstMAC: locVX.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv6},
			&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: net.IP(locVX.RemoteIP.AsSlice()), DstIP: net.IP(locVX.LocalIP.AsSlice())},
			&layers.UDP{SrcPort: layers.UDPPort(65535 - i), DstPort: 4789},
			&layers.VXLAN{ValidIDFlag: true, VNI: uint32(locVX.VXLAN)},
			&layers.Ethernet{SrcMAC: locVX.InnerRemote.HardwareAddr, DstMAC: locVX.InnerLocal.HardwareAddr, EthernetType: an.EtherTypeNDN},
			makeRxFrame("VX", i),
		)
		assert.NoError(e)
		iface.TxBurst(faceVX.ID(), makeTxBurst("VX", i))
	}

	time.Sleep(10 * time.Millisecond)

	assert.EqualValues(500, faceEther.Counters().RxInterests)
	assert.EqualValues(500, faceUDP4.Counters().RxInterests)
	assert.EqualValues(500, faceUDP4p1.Counters().RxInterests)
	assert.EqualValues(500, faceUDP6.Counters().RxInterests)
	assert.EqualValues(500, faceVX.Counters().RxInterests)

	assert.EqualValues(500, txEther.Load())
	assert.EqualValues(500, txUDP4.Load())
	assert.EqualValues(500, txUDP4p1.Load())
	assert.EqualValues(500, txUDP6.Load())
	assert.EqualValues(500, txVX.Load())
	assert.Less(int(txOther.Load()), 50)
}

func TestXDP(t *testing.T) {
	_, require := makeAR(t)
	xdpProgram, e := bpf.XDP.Find("redir")
	require.NoError(e)

	testPortTAP(t, func(tunName string) ethnetif.Config {
		return ethnetif.Config{
			Driver:     ethnetif.DriverXDP,
			Netif:      tunName,
			XDPProgram: xdpProgram,
		}
	})
}

func TestAfPacket(t *testing.T) {
	testPortTAP(t, func(tunName string) ethnetif.Config {
		return ethnetif.Config{
			Driver: ethnetif.DriverAfPacket,
			Netif:  tunName,
		}
	})
}
