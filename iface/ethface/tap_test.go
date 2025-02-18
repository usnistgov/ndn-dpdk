package ethface_test

import (
	"fmt"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
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

// TapFixture organizes a TAP netif that simulates hardware and an ethport.Port linked to the TAP netif.
type TapFixture struct {
	t    testing.TB
	Intf *water.Interface
	Port *ethport.Port
}

func (tap *TapFixture) AddFace(loc ethport.Locator) iface.Face {
	_, require := makeAR(tap.t)
	face, e := loc.CreateFace()
	require.NoError(e)
	tap.t.Cleanup(func() { must.Close(face) })
	return face
}

// WriteToFromLayers causes the TAP netif to receive an Ethernet frame.
func (tap *TapFixture) WriteToFromLayers(hdrs ...gopacket.SerializableLayer) {
	assert, _ := makeAR(tap.t)
	_, e := writeToFromLayers(tap.Intf, hdrs...)
	assert.NoError(e)
}

func newTapFixture(
	t testing.TB,
	makeNetifConfig func(ifname string) ethnetif.Config,
) *TapFixture {
	_, require := makeAR(t)

	cfg := water.Config{DeviceType: water.TAP}
	intf, e := water.New(cfg)
	require.NoError(e)
	t.Cleanup(func() { must.Close(intf) })

	link, e := netlink.LinkByName(intf.Name())
	require.NoError(e)
	e = netlink.LinkSetHardwareAddr(link, macaddr.MakeRandomUnicast())
	require.NoError(e)

	port, e := ethport.New(ethport.Config{
		Config: makeNetifConfig(intf.Name()),
	})
	require.NoError(e)
	t.Cleanup(func() { must.Close(port) })

	return &TapFixture{
		t:    t,
		Intf: intf,
		Port: port,
	}
}

func newTapFixtureAfPacket(t testing.TB) *TapFixture {
	return newTapFixture(t, func(ifname string) ethnetif.Config {
		return ethnetif.Config{
			Driver: ethnetif.DriverAfPacket,
			Netif:  ifname,
		}
	})
}

func makeRxFrame(prefix string, i int) gopacket.SerializableLayer {
	interest := ndn.MakeInterest(fmt.Sprintf("/RX/%s/%d", prefix, i))
	return &ndnlayer.NDN{Packet: interest.ToPacket()}
}

func makeTxBurst(prefix string, i int) []*ndni.Packet {
	return []*ndni.Packet{makeInterest(fmt.Sprintf("/TX/%s/%d", prefix, i))}
}

func testPortTap(tap *TapFixture) {
	assert, _ := makeAR(tap.t)

	var locEther ethface.EtherLocator
	locEther.Local.HardwareAddr = tap.Port.EthDev().HardwareAddr()
	locEther.Remote.Set("02:00:00:00:00:02")
	locEther.VLAN = 1987
	faceEther := tap.AddFace(locEther)

	var locUDP4 ethface.UDPLocator
	locUDP4.EtherLocator = locEther
	locUDP4.LocalIP, locUDP4.LocalUDP = netip.MustParseAddr("192.168.2.1"), 6363
	locUDP4.RemoteIP, locUDP4.RemoteUDP = netip.MustParseAddr("192.168.2.2"), 6363
	faceUDP4 := tap.AddFace(locUDP4)

	locUDP4p1 := locUDP4
	locUDP4p1.LocalUDP, locUDP4p1.RemoteUDP = 16363, 26363
	faceUDP4p1 := tap.AddFace(locUDP4p1)

	locUDP6 := locUDP4
	locUDP6.VLAN = 0
	locUDP6.LocalIP = netip.MustParseAddr("fde0:fd0a:3557:a8c7:db87:639f:9bd2:0001")
	locUDP6.RemoteIP = netip.MustParseAddr("fde0:fd0a:3557:a8c7:db87:639f:9bd2:0002")
	faceUDP6 := tap.AddFace(locUDP6)

	var locVX ethface.VxlanLocator
	locVX.IPLocator = locUDP6.IPLocator
	locVX.VXLAN = 0x887700
	locVX.InnerLocal.Set("02:00:00:00:01:01")
	locVX.InnerRemote.Set("02:00:00:00:01:02")
	faceVX := tap.AddFace(locVX)

	var locGTP8 ethface.GtpLocator
	locGTP8.IPLocator = locUDP4.IPLocator
	locGTP8.VLAN = 0
	locGTP8.UlTEID, locGTP8.DlTEID = 0x10000008, 0x20000008
	locGTP8.UlQFI, locGTP8.DlQFI = 2, 12
	locGTP8.InnerLocalIP = netip.MustParseAddr("192.168.60.3")
	locGTP8.InnerRemoteIP = netip.MustParseAddr("192.168.60.4")
	faceGTP8 := tap.AddFace(locGTP8)

	locGTP9 := locGTP8
	locGTP9.UlTEID, locGTP9.DlTEID = 0x10000009, 0x20000009
	faceGTP9 := tap.AddFace(locGTP9)

	var txEther, txUDP4, txUDP4p1, txUDP6, txVX, txGTP8, txGTP9, txOther atomic.Int32
	go func() {
		buf := make([]byte, tap.Port.EthDev().MTU())
		for {
			n, e := tap.Intf.Read(buf)
			if e != nil {
				break
			}

			classify, isV4, gtp := &txOther, false, 0
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
					switch {
					case int(l.SrcPort) == locUDP4p1.LocalUDP:
						classify = &txUDP4p1
					case isV4:
						switch gtp {
						case 0:
							classify = &txUDP4
						case 8:
							classify = &txGTP8
						case 9:
							classify = &txGTP9
						}
					default:
						classify = &txUDP6
					}
				case *layers.VXLAN:
					classify = &txVX
				case *layers.GTPv1U:
					gtp = int(l.TEID & 0xFF)
				}
			}
			classify.Add(1)
		}
	}()

	for i := range 500 {
		time.Sleep(10 * time.Millisecond)

		tap.WriteToFromLayers(
			&layers.Ethernet{SrcMAC: locEther.Remote.HardwareAddr, DstMAC: locEther.Local.HardwareAddr, EthernetType: layers.EthernetTypeDot1Q},
			&layers.Dot1Q{VLANIdentifier: uint16(locEther.VLAN), Type: an.EtherTypeNDN},
			makeRxFrame("Ether", i),
		)
		iface.TxBurst(faceEther.ID(), makeTxBurst("Ether", i))

		tap.WriteToFromLayers(
			&layers.Ethernet{SrcMAC: locUDP4.Remote.HardwareAddr, DstMAC: locUDP4.Local.HardwareAddr, EthernetType: layers.EthernetTypeDot1Q},
			&layers.Dot1Q{VLANIdentifier: uint16(locUDP4.VLAN), Type: layers.EthernetTypeIPv4},
			&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: locUDP4.RemoteIP.AsSlice(), DstIP: locUDP4.LocalIP.AsSlice()},
			&layers.UDP{SrcPort: layers.UDPPort(locUDP4.RemoteUDP), DstPort: layers.UDPPort(locUDP4.LocalUDP)},
			makeRxFrame("UDP4", i),
		)
		iface.TxBurst(faceUDP4.ID(), makeTxBurst("UDP4", i))

		tap.WriteToFromLayers(
			&layers.Ethernet{SrcMAC: locUDP4p1.Remote.HardwareAddr, DstMAC: locUDP4p1.Local.HardwareAddr, EthernetType: layers.EthernetTypeDot1Q},
			&layers.Dot1Q{Priority: 1, VLANIdentifier: uint16(locUDP4p1.VLAN), Type: layers.EthernetTypeIPv4},
			&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: locUDP4p1.RemoteIP.AsSlice(), DstIP: locUDP4p1.LocalIP.AsSlice()},
			&layers.UDP{SrcPort: layers.UDPPort(locUDP4p1.RemoteUDP), DstPort: layers.UDPPort(locUDP4p1.LocalUDP)},
			makeRxFrame("UDP4p1", i),
		)
		iface.TxBurst(faceUDP4p1.ID(), makeTxBurst("UDP4p1", i))

		tap.WriteToFromLayers(
			&layers.Ethernet{SrcMAC: locUDP6.Remote.HardwareAddr, DstMAC: locUDP6.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv6},
			&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: locUDP6.RemoteIP.AsSlice(), DstIP: locUDP6.LocalIP.AsSlice()},
			&layers.UDP{SrcPort: layers.UDPPort(locUDP6.RemoteUDP), DstPort: layers.UDPPort(locUDP6.LocalUDP)},
			makeRxFrame("UDP6", i),
		)
		iface.TxBurst(faceUDP6.ID(), makeTxBurst("UDP6", i))

		tap.WriteToFromLayers(
			&layers.Ethernet{SrcMAC: locVX.Remote.HardwareAddr, DstMAC: locVX.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv6},
			&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: locVX.RemoteIP.AsSlice(), DstIP: locVX.LocalIP.AsSlice()},
			&layers.UDP{SrcPort: layers.UDPPort(65535 - i), DstPort: 4789},
			&layers.VXLAN{ValidIDFlag: true, VNI: uint32(locVX.VXLAN)},
			&layers.Ethernet{SrcMAC: locVX.InnerRemote.HardwareAddr, DstMAC: locVX.InnerLocal.HardwareAddr, EthernetType: an.EtherTypeNDN},
			makeRxFrame("VX", i),
		)
		iface.TxBurst(faceVX.ID(), makeTxBurst("VX", i))

		tap.WriteToFromLayers(
			&layers.Ethernet{SrcMAC: locGTP8.Remote.HardwareAddr, DstMAC: locGTP8.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv4},
			&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: locGTP8.RemoteIP.AsSlice(), DstIP: locGTP8.LocalIP.AsSlice()},
			&layers.UDP{SrcPort: 2152, DstPort: 2152},
			makeGTPv1U(uint32(locGTP8.UlTEID), 1, uint8(locGTP8.UlQFI)),
			&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: locGTP8.InnerRemoteIP.AsSlice(), DstIP: locGTP8.InnerLocalIP.AsSlice()},
			&layers.UDP{SrcPort: 6363, DstPort: 6363},
			makeRxFrame("GTP8", i),
		)
		iface.TxBurst(faceGTP8.ID(), makeTxBurst("GTP8", i))

		tap.WriteToFromLayers(
			&layers.Ethernet{SrcMAC: locGTP9.Remote.HardwareAddr, DstMAC: locGTP9.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv4},
			&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: locGTP9.RemoteIP.AsSlice(), DstIP: locGTP9.LocalIP.AsSlice()},
			&layers.UDP{SrcPort: 2152, DstPort: 2152},
			makeGTPv1U(uint32(locGTP9.UlTEID), 1, uint8(locGTP9.UlQFI)),
			&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: locGTP9.InnerRemoteIP.AsSlice(), DstIP: locGTP9.InnerLocalIP.AsSlice()},
			&layers.UDP{SrcPort: 6363, DstPort: 6363},
			makeRxFrame("GTP9", i),
		)
		iface.TxBurst(faceGTP9.ID(), makeTxBurst("GTP9", i))
	}

	time.Sleep(10 * time.Millisecond)

	assert.EqualValues(500, faceEther.Counters().RxInterests)
	assert.EqualValues(500, faceUDP4.Counters().RxInterests)
	assert.EqualValues(500, faceUDP4p1.Counters().RxInterests)
	assert.EqualValues(500, faceUDP6.Counters().RxInterests)
	assert.EqualValues(500, faceVX.Counters().RxInterests)
	assert.EqualValues(500, faceGTP8.Counters().RxInterests)
	assert.EqualValues(500, faceGTP9.Counters().RxInterests)

	assert.EqualValues(500, txEther.Load())
	assert.EqualValues(500, txUDP4.Load())
	assert.EqualValues(500, txUDP4p1.Load())
	assert.EqualValues(500, txUDP6.Load())
	assert.EqualValues(500, txVX.Load())
	assert.EqualValues(500, txGTP8.Load())
	assert.EqualValues(500, txGTP9.Load())
	assert.Less(int(txOther.Load()), 50)
}

func TestXDP(t *testing.T) {
	_, require := makeAR(t)
	xdpProgram, e := bpf.XDP.Find("redir")
	require.NoError(e)

	tap := newTapFixture(t, func(ifname string) ethnetif.Config {
		return ethnetif.Config{
			Driver:     ethnetif.DriverXDP,
			Netif:      ifname,
			XDPProgram: xdpProgram,
		}
	})
	testPortTap(tap)
}

func TestAfPacket(t *testing.T) {
	tap := newTapFixtureAfPacket(t)
	testPortTap(tap)
}
