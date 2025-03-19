package ethface_test

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/songgao/water"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/core/pciaddr"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethnetif"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/iface/ifacetestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/ethertransport"
	"github.com/usnistgov/ndn-dpdk/ndn/ndnlayer"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/vishvananda/netlink"
	"go4.org/must"
)

const dfltEpsilon = 0.02

// PortRemoteFixture tests a local port with a connected "remote" network interface.
// The local port is a DPDK EthDev using AfPacket, XDP, or PCI driver.
// The remote network interface is either a TAP file descriptor associated with the local port,
// or another Ethernet adapter (often a PCI Virtual Function) connected with the local port.
type PortRemoteFixture struct {
	t               testing.TB
	RemoteNetif     *ethnetif.NetIntf // available if remote netif is a Virtual Function or similar
	RemoteMAC       net.HardwareAddr
	RemoteIntf      io.ReadWriteCloser // TAP or AF_PACKET file descriptor
	LocalPort       *ethport.Port
	LocalMAC        net.HardwareAddr
	PassthruNetif   *ethnetif.NetIntf
	DiagFaces       func() // invoked after face creation and before traffic generation
	RxLossTolerance float64
	TxLossTolerance float64
}

// AddFace creates a face in the local port.
func (prf *PortRemoteFixture) AddFace(loc ethport.Locator) iface.Face {
	_, require := makeAR(prf.t)
	face, e := loc.CreateFace()
	require.NoError(e)
	return face
}

// SetupPassthruNetif locates PassthruNetif and assigns IP addresses.
// Panics if the local port does not have a pass-through face.
func (prf *PortRemoteFixture) SetupPassthruNetif(addrs ...netip.Prefix) {
	_, require := makeAR(prf.t)

	var e error
	prf.PassthruNetif, e = ethnetif.NetIntfByName(ethport.MakePassthruTapName(prf.LocalPort.EthDev()))
	require.NoError(e)

	require.NoError(prf.PassthruNetif.EnsureLinkUp())

	for _, addr := range addrs {
		require.NoError(prf.PassthruNetif.SetIP(addr))
	}
}

// CapturePassthru opens AF_PACKET socket on the PassthruNetif.
func (prf *PortRemoteFixture) CapturePassthru() *ethertransport.ConnHandle {
	_, require := makeAR(prf.t)

	netif, e := net.InterfaceByIndex(prf.PassthruNetif.Index)
	require.NoError(e)

	hdl, e := ethertransport.NewConnHandle(netif, 0)
	require.NoError(e)
	prf.t.Cleanup(func() { must.Close(hdl) })

	return hdl
}

// OverrideMAC returns prf.RemoteMAC if non-empty, otherwise returns input MAC.
//
// In tests that support both PCI Virtual Function and promisc TAP netifs, the TAP test cases
// can set prf.RemoteMAC to nil and then pass random/arbitrary MAC address through this function.
func (prf *PortRemoteFixture) OverrideMAC(r net.HardwareAddr) net.HardwareAddr {
	if len(prf.RemoteMAC) == 0 {
		return r
	}
	return prf.RemoteMAC
}

// RemoteWrite writes an Ethernet frame via the remote intf.
// This typically causes the local port to receive the frame.
func (prf *PortRemoteFixture) RemoteWrite(hdrs ...gopacket.SerializableLayer) {
	assert, _ := makeAR(prf.t)
	b, discard := packetFromLayers(hdrs...)
	defer discard()
	_, e := prf.RemoteIntf.Write(b)
	assert.NoError(e)
}

// MakeRxFrame makes the NDN layer of a received packet.
func (PortRemoteFixture) MakeRxFrame(prefix string, i int) gopacket.SerializableLayer {
	interest := ndn.MakeInterest(fmt.Sprintf("/RX/%s/%d", prefix, i))
	return &ndnlayer.NDN{Packet: interest.ToPacket()}
}

// MakeTxBurst makes a burst of TX packets.
func (PortRemoteFixture) MakeTxBurst(prefix string, i int) []*ndni.Packet {
	return []*ndni.Packet{makeInterest(fmt.Sprintf("/TX/%s/%d", prefix, i))}
}

// NewPortRemoteFixture creates a PortRemoteFixture.
//
//	t: *testing.T
//	remoteIfname: remote netif name; empty string to create TAP netif.
//	localIfname: local netif name; empty string to use TAP netif.
//	configPort: callback to create ethport.Config from local netif name. Default to AF_PACKET driver.
func NewPortRemoteFixture(
	t testing.TB,
	remoteIfname, localIfname string,
	configPort func(ifname string) ethport.Config,
) *PortRemoteFixture {
	_, require := makeAR(t)
	t.Cleanup(ifacetestenv.ClearFacesLCores)
	prf := &PortRemoteFixture{
		t:               t,
		DiagFaces:       func() {},
		RxLossTolerance: dfltEpsilon,
		TxLossTolerance: dfltEpsilon,
	}

	if remoteIfname == "" {
		require.Empty(localIfname)
		prf.RemoteMAC = macaddr.MakeRandomUnicast()

		intf, e := water.New(water.Config{DeviceType: water.TAP})
		require.NoError(e)
		t.Cleanup(func() { must.Close(intf) })
		prf.RemoteIntf = intf
		localIfname = intf.Name()

		link, e := netlink.LinkByName(intf.Name())
		require.NoError(e)
		e = netlink.LinkSetHardwareAddr(link, macaddr.MakeRandomUnicast())
		require.NoError(e)
	} else {
		link, e := ethnetif.NetIntfByName(remoteIfname)
		require.NoError(e)
		e = link.EnsureLinkUp()
		require.NoError(e)
		link.SetOffload("rx-vlan-filter", false)
		if link.Promisc == 0 {
			if e = netlink.SetPromiscOn(link.Link); e != nil {
				t.Logf("netlink.SetPromiscOn(%s) error %v", link.Name, e)
			} else {
				t.Cleanup(func() { netlink.SetPromiscOff(link.Link) })
			}
		}
		prf.RemoteNetif, prf.RemoteMAC = link, link.HardwareAddr

		netif, e := net.InterfaceByIndex(link.Index)
		require.NoError(e)
		prf.RemoteIntf, e = ethertransport.NewConnHandle(netif, 0)
		require.NoError(e)
		t.Cleanup(func() { must.Close(prf.RemoteIntf) })
	}

	if configPort == nil {
		configPort = func(ifname string) ethport.Config {
			return ethport.Config{
				Config: ethnetif.Config{
					Driver: ethnetif.DriverAfPacket,
					Netif:  ifname,
				},
			}
		}
	}
	port, e := ethport.New(configPort(localIfname))
	require.NoError(e)
	prf.LocalPort = port
	prf.LocalMAC = port.EthDev().HardwareAddr()

	return prf
}

func configPortXDP(ifname string) ethport.Config {
	return ethport.Config{
		Config: ethnetif.Config{
			Driver: ethnetif.DriverXDP,
			Netif:  ifname,
		},
	}
}

// testPortRemote tests faces and locators between a remote and a local face.
//
//	selections:
//	- If empty, all locators are enabled.
//	- If first item is "-", specified locators are disabled.
//	- If first item is not "-", specified locators are enabled.
//	- if "+rss" exists, VXLAN faces have two queues.
func testPortRemote(prf *PortRemoteFixture, selections []string) {
	assert, _ := makeAR(prf.t)

	type portRemoteFaceRecord struct {
		Title string
		Face  iface.Face
		TxCnt atomic.Int32
	}
	faces := map[string]*portRemoteFaceRecord{}
	addFaceIfEnabled := func(title string, loc ethport.Locator) {
		switch {
		case len(selections) == 0,
			selections[0] == "-" && !slices.Contains(selections, title),
			selections[0] != "-" && slices.Contains(selections, title):
		default:
			return
		}

		if loc.Scheme() == "vxlan" && slices.Contains(selections, "+rss") {
			locVx := loc.(ethface.VxlanLocator)
			locVx.NRxQueues = 2
			loc = locVx
		}

		prf.t.Logf("creating face %s", title)
		faces[title] = &portRemoteFaceRecord{
			Face: prf.AddFace(loc),
		}
	}

	var locPassthru ethface.PassthruLocator
	locPassthru.Local.HardwareAddr = prf.LocalMAC
	addFaceIfEnabled("passthru", locPassthru)

	var locEther ethface.EtherLocator
	locEther.Local.HardwareAddr = prf.LocalMAC
	locEther.Remote.HardwareAddr = prf.RemoteMAC
	addFaceIfEnabled("ether", locEther)

	locEtherMcast := locEther
	locEtherMcast.Remote.HardwareAddr = ethertransport.MulticastAddressNDN
	addFaceIfEnabled("ether-mcast", locEtherMcast)

	locVlan := locEther
	locVlan.VLAN = 1987
	addFaceIfEnabled("vlan", locVlan)

	var locUDP4 ethface.UDPLocator
	locUDP4.EtherLocator = locEther
	locUDP4.LocalIP, locUDP4.LocalUDP = netip.MustParseAddr("192.168.2.1"), 6363
	locUDP4.RemoteIP, locUDP4.RemoteUDP = netip.MustParseAddr("192.168.2.2"), 6363
	addFaceIfEnabled("udp4", locUDP4)

	locUDP4p1 := locUDP4
	locUDP4p1.LocalUDP, locUDP4p1.RemoteUDP = 16363, 26363
	addFaceIfEnabled("udp4p1", locUDP4p1)

	locVlanUDP4 := locUDP4
	locVlanUDP4.EtherLocator = locVlan
	addFaceIfEnabled("vlan-udp4", locVlanUDP4)

	locUDP6 := locUDP4
	locUDP6.LocalIP = netip.MustParseAddr("fde0:fd0a:3557:a8c7:db87:639f:9bd2:0001")
	locUDP6.RemoteIP = netip.MustParseAddr("fde0:fd0a:3557:a8c7:db87:639f:9bd2:0002")
	addFaceIfEnabled("udp6", locUDP6)

	locVlanUDP6 := locUDP6
	locVlanUDP6.EtherLocator = locVlan
	addFaceIfEnabled("vlan-udp6", locVlanUDP6)

	var locVx4 ethface.VxlanLocator
	locVx4.IPLocator = locUDP4.IPLocator
	locVx4.VXLAN = 0x887700
	locVx4.InnerLocal.Set("02:00:00:00:01:01")
	locVx4.InnerRemote.Set("02:00:00:00:01:02")
	addFaceIfEnabled("vx4", locVx4)

	locVx6 := locVx4
	locVx6.IPLocator = locUDP6.IPLocator
	addFaceIfEnabled("vx6", locVx6)

	var locGtp8 ethface.GtpLocator
	locGtp8.IPLocator = locUDP4.IPLocator
	locGtp8.UlTEID, locGtp8.DlTEID = 0x10000008, 0x20000008
	locGtp8.UlQFI, locGtp8.DlQFI = 2, 12
	locGtp8.InnerLocalIP = netip.MustParseAddr("192.168.60.3")
	locGtp8.InnerRemoteIP = netip.MustParseAddr("192.168.60.4")
	addFaceIfEnabled("gtp8", locGtp8)

	locGtp9 := locGtp8
	locGtp9.UlTEID, locGtp9.DlTEID = 0x10000009, 0x20000009
	addFaceIfEnabled("gtp9", locGtp9)

	if _, ok := faces["passthru"]; ok {
		prf.SetupPassthruNetif(netip.PrefixFrom(locUDP4.LocalIP, 24))
	}
	passthruArpOnly := slices.Contains(selections, "+arp-only")

	prf.DiagFaces()

	// Observe packets transmitted by the local port by receiving them on the remote netif.
	var txOther atomic.Int32
	go func() {
		buf := make([]byte, 9200)
	DROP:
		for {
			n, e := prf.RemoteIntf.Read(buf)
			if e != nil {
				break
			}

			pkt := buf[:n]
			parsed := gopacket.NewPacket(pkt, layers.LayerTypeEthernet, gopacket.NoCopy)
			classify, isV4, isVlan, gtp := "", false, false, 0
			for i, l := range parsed.Layers() {
				switch l := l.(type) {
				case *layers.Ethernet:
					switch {
					case i > 0:
					case !bytes.Equal(prf.LocalMAC, l.SrcMAC):
						continue DROP
					case l.EthernetType == an.EtherTypeNDN:
						classify = "ether"
						if macaddr.IsMulticast(l.DstMAC) {
							classify = "ether-mcast"
						}
					}
				case *layers.Dot1Q:
					isVlan = true
					if l.Type == an.EtherTypeNDN {
						classify = "vlan"
					}
				case *layers.ARP:
					classify = "passthru"
				case *layers.IPv4:
					isV4 = true
				case *layers.ICMPv4:
					classify = "passthru"
				case *layers.UDP:
					switch {
					case int(l.SrcPort) == locUDP4p1.LocalUDP:
						classify = "udp4p1"
					case isV4:
						switch gtp {
						case 0:
							if isVlan {
								classify = "vlan-udp4"
							} else {
								classify = "udp4"
							}
						case 8:
							classify = "gtp8"
						case 9:
							classify = "gtp9"
						}
					default:
						if isVlan {
							classify = "vlan-udp6"
						} else {
							classify = "udp6"
						}
					}
				case *layers.VXLAN:
					if isV4 {
						classify = "vx4"
					} else {
						classify = "vx6"
					}
				case *layers.GTPv1U:
					gtp = int(l.TEID & 0xFF)
				}
			}
			if rec := faces[classify]; rec == nil {
				txOther.Add(1)
			} else {
				rec.TxCnt.Add(1)
			}
		}
	}()

	// Transmit packets from the remote netif to be received by the local port.
	// Transmit packets from a local face to be transmitted by the local port.
	nRounds := 500
	for i := range nRounds {
		time.Sleep(10 * time.Millisecond)

		if loc, rec := locUDP4, faces["passthru"]; rec != nil {
			if i%5 == 0 || passthruArpOnly {
				prf.RemoteWrite(
					makeARP(loc.Remote.HardwareAddr, loc.RemoteIP.AsSlice(), nil, loc.LocalIP.AsSlice())...,
				)
				// kernel will send ARP reply
			} else {
				prf.RemoteWrite(
					&layers.Ethernet{SrcMAC: loc.Remote.HardwareAddr, DstMAC: loc.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv4},
					&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolICMPv4, SrcIP: locUDP4.RemoteIP.AsSlice(), DstIP: locUDP4.LocalIP.AsSlice()},
					&layers.ICMPv4{TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0), Id: 1, Seq: uint16(i)},
				)
				// kernel will send ICMPv4 reply
			}
		}

		if loc, rec := locEther, faces["ether"]; rec != nil {
			prf.RemoteWrite(
				&layers.Ethernet{SrcMAC: loc.Remote.HardwareAddr, DstMAC: loc.Local.HardwareAddr, EthernetType: an.EtherTypeNDN},
				prf.MakeRxFrame("ether", i),
			)
			iface.TxBurst(rec.Face.ID(), prf.MakeTxBurst("ether", i))
		}

		if loc, rec := locEtherMcast, faces["ether-mcast"]; rec != nil {
			prf.RemoteWrite(
				&layers.Ethernet{SrcMAC: prf.RemoteMAC, DstMAC: loc.Remote.HardwareAddr, EthernetType: an.EtherTypeNDN},
				prf.MakeRxFrame("ether-mcast", i),
			)
			iface.TxBurst(rec.Face.ID(), prf.MakeTxBurst("ether-mcast", i))
		}

		if loc, rec := locVlan, faces["vlan"]; rec != nil {
			prf.RemoteWrite(
				&layers.Ethernet{SrcMAC: loc.Remote.HardwareAddr, DstMAC: loc.Local.HardwareAddr, EthernetType: layers.EthernetTypeDot1Q},
				&layers.Dot1Q{VLANIdentifier: uint16(loc.VLAN), Type: an.EtherTypeNDN},
				prf.MakeRxFrame("vlan", i),
			)
			iface.TxBurst(rec.Face.ID(), prf.MakeTxBurst("vlan", i))
		}

		if loc, rec := locUDP4, faces["udp4"]; rec != nil {
			prf.RemoteWrite(
				&layers.Ethernet{SrcMAC: loc.Remote.HardwareAddr, DstMAC: loc.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv4},
				&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: loc.RemoteIP.AsSlice(), DstIP: loc.LocalIP.AsSlice()},
				&layers.UDP{SrcPort: layers.UDPPort(loc.RemoteUDP), DstPort: layers.UDPPort(loc.LocalUDP)},
				prf.MakeRxFrame("udp4", i),
			)
			iface.TxBurst(rec.Face.ID(), prf.MakeTxBurst("udp4", i))
		}

		if loc, rec := locUDP4p1, faces["udp4p1"]; rec != nil {
			prf.RemoteWrite(
				&layers.Ethernet{SrcMAC: loc.Remote.HardwareAddr, DstMAC: loc.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv4},
				&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: loc.RemoteIP.AsSlice(), DstIP: loc.LocalIP.AsSlice()},
				&layers.UDP{SrcPort: layers.UDPPort(loc.RemoteUDP), DstPort: layers.UDPPort(loc.LocalUDP)},
				prf.MakeRxFrame("udp4p1", i),
			)
			iface.TxBurst(rec.Face.ID(), prf.MakeTxBurst("udp4p1", i))
		}

		if loc, rec := locVlanUDP4, faces["vlan-udp4"]; rec != nil {
			prf.RemoteWrite(
				&layers.Ethernet{SrcMAC: loc.Remote.HardwareAddr, DstMAC: loc.Local.HardwareAddr, EthernetType: layers.EthernetTypeDot1Q},
				&layers.Dot1Q{VLANIdentifier: uint16(loc.VLAN), Type: layers.EthernetTypeIPv4},
				&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: loc.RemoteIP.AsSlice(), DstIP: loc.LocalIP.AsSlice()},
				&layers.UDP{SrcPort: layers.UDPPort(loc.RemoteUDP), DstPort: layers.UDPPort(loc.LocalUDP)},
				prf.MakeRxFrame("vlan-udp4", i),
			)
			iface.TxBurst(rec.Face.ID(), prf.MakeTxBurst("vlan-udp4", i))
		}

		if loc, rec := locUDP6, faces["udp6"]; rec != nil {
			prf.RemoteWrite(
				&layers.Ethernet{SrcMAC: loc.Remote.HardwareAddr, DstMAC: loc.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv6},
				&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: loc.RemoteIP.AsSlice(), DstIP: loc.LocalIP.AsSlice()},
				&layers.UDP{SrcPort: layers.UDPPort(loc.RemoteUDP), DstPort: layers.UDPPort(loc.LocalUDP)},
				prf.MakeRxFrame("udp6", i),
			)
			iface.TxBurst(rec.Face.ID(), prf.MakeTxBurst("udp6", i))
		}

		if loc, rec := locVlanUDP6, faces["vlan-udp6"]; rec != nil {
			prf.RemoteWrite(
				&layers.Ethernet{SrcMAC: loc.Remote.HardwareAddr, DstMAC: loc.Local.HardwareAddr, EthernetType: layers.EthernetTypeDot1Q},
				&layers.Dot1Q{VLANIdentifier: uint16(loc.VLAN), Type: layers.EthernetTypeIPv6},
				&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: loc.RemoteIP.AsSlice(), DstIP: loc.LocalIP.AsSlice()},
				&layers.UDP{SrcPort: layers.UDPPort(loc.RemoteUDP), DstPort: layers.UDPPort(loc.LocalUDP)},
				prf.MakeRxFrame("vlan-udp6", i),
			)
			iface.TxBurst(rec.Face.ID(), prf.MakeTxBurst("vlan-udp6", i))
		}

		if loc, rec := locVx4, faces["vx4"]; rec != nil {
			prf.RemoteWrite(
				&layers.Ethernet{SrcMAC: loc.Remote.HardwareAddr, DstMAC: loc.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv4},
				&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: loc.RemoteIP.AsSlice(), DstIP: loc.LocalIP.AsSlice()},
				&layers.UDP{SrcPort: layers.UDPPort(65535 - i), DstPort: 4789},
				&layers.VXLAN{ValidIDFlag: true, VNI: uint32(loc.VXLAN)},
				&layers.Ethernet{SrcMAC: loc.InnerRemote.HardwareAddr, DstMAC: loc.InnerLocal.HardwareAddr, EthernetType: an.EtherTypeNDN},
				prf.MakeRxFrame("vx4", i),
			)
			iface.TxBurst(rec.Face.ID(), prf.MakeTxBurst("vx4", i))
		}

		if loc, rec := locVx6, faces["vx6"]; rec != nil {
			prf.RemoteWrite(
				&layers.Ethernet{SrcMAC: loc.Remote.HardwareAddr, DstMAC: loc.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv6},
				&layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, SrcIP: loc.RemoteIP.AsSlice(), DstIP: loc.LocalIP.AsSlice()},
				&layers.UDP{SrcPort: layers.UDPPort(65535 - i), DstPort: 4789},
				&layers.VXLAN{ValidIDFlag: true, VNI: uint32(loc.VXLAN)},
				&layers.Ethernet{SrcMAC: loc.InnerRemote.HardwareAddr, DstMAC: loc.InnerLocal.HardwareAddr, EthernetType: an.EtherTypeNDN},
				prf.MakeRxFrame("vx6", i),
			)
			iface.TxBurst(rec.Face.ID(), prf.MakeTxBurst("vx6", i))
		}

		if loc, rec := locGtp8, faces["gtp8"]; rec != nil {
			prf.RemoteWrite(
				&layers.Ethernet{SrcMAC: loc.Remote.HardwareAddr, DstMAC: loc.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv4},
				&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: loc.RemoteIP.AsSlice(), DstIP: loc.LocalIP.AsSlice()},
				&layers.UDP{SrcPort: 2152, DstPort: 2152},
				makeGTPv1U(uint32(loc.UlTEID), 1, uint8(loc.UlQFI)),
				&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: loc.InnerRemoteIP.AsSlice(), DstIP: loc.InnerLocalIP.AsSlice()},
				&layers.UDP{SrcPort: 6363, DstPort: 6363},
				prf.MakeRxFrame("gtp8", i),
			)
			iface.TxBurst(rec.Face.ID(), prf.MakeTxBurst("gtp8", i))
		}

		if loc, rec := locGtp9, faces["gtp9"]; rec != nil {
			prf.RemoteWrite(
				&layers.Ethernet{SrcMAC: loc.Remote.HardwareAddr, DstMAC: loc.Local.HardwareAddr, EthernetType: layers.EthernetTypeIPv4},
				&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: loc.RemoteIP.AsSlice(), DstIP: loc.LocalIP.AsSlice()},
				&layers.UDP{SrcPort: 2152, DstPort: 2152},
				makeGTPv1U(uint32(loc.UlTEID), 1, uint8(loc.UlQFI)),
				&layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: loc.InnerRemoteIP.AsSlice(), DstIP: loc.InnerLocalIP.AsSlice()},
				&layers.UDP{SrcPort: 6363, DstPort: 6363},
				prf.MakeRxFrame("gtp9", i),
			)
			iface.TxBurst(rec.Face.ID(), prf.MakeTxBurst("gtp9", i))
		}
	}
	time.Sleep(10 * time.Millisecond)

	for title, rec := range faces {
		uTolerance := 0.0
		if title == "passthru" {
			uTolerance = 0.02
		}
		testenv.AtOrAround(assert, nRounds, rec.Face.Counters().RxInterests, prf.RxLossTolerance, uTolerance, title)
		testenv.AtOrAround(assert, nRounds, rec.TxCnt.Load(), prf.TxLossTolerance, uTolerance, title)
	}
	assert.Less(int(txOther.Load()), nRounds/10)
}

func TestTapXDP(t *testing.T) {
	prf := NewPortRemoteFixture(t, "", "", configPortXDP)
	testPortRemote(prf, []string{"-", "passthru"})
}

func TestTapAfPacket(t *testing.T) {
	prf := NewPortRemoteFixture(t, "", "", nil)
	testPortRemote(prf, nil)
}

type vfPairEnv struct {
	t               testing.TB
	RemoteIfname    string
	LocalIfname     string
	LocalPCI        *pciaddr.PCIAddress
	RxFlowQueues    int
	RxLossTolerance float64
	TxLossTolerance float64
}

func (env *vfPairEnv) MakePrf(configPort func(ifname string) ethport.Config) *PortRemoteFixture {
	prf := NewPortRemoteFixture(env.t, env.RemoteIfname, env.LocalIfname, configPort)
	prf.DiagFaces = func() {
		dump, e := ethdev.GetFlowDump(prf.LocalPort.EthDev())
		env.t.Logf("FlowDump: err=%v\n%s", e, dump)
	}
	prf.RxLossTolerance, prf.TxLossTolerance = env.RxLossTolerance, env.TxLossTolerance
	return prf
}

// ConfigPortPCI can be passed as NewPortRemoteFixture configPort argument to configure a port with PCI driver.
//
// parseVfTestEnv initializes env.RxFlowQueues to non-zero, which would create the port with RxFlow.
// Set env.RxFlowQueues to zero, in order to create the port with RxTable.
func (env *vfPairEnv) ConfigPortPCI(ifname string) ethport.Config {
	netifConfig := ethnetif.Config{
		Driver: ethnetif.DriverPCI,
	}
	if env.LocalPCI == nil {
		netifConfig.Netif = ifname
	} else {
		netifConfig.PCIAddr = env.LocalPCI
	}

	return ethport.Config{
		Config:       netifConfig,
		RxQueueSize:  512,
		TxQueueSize:  512,
		RxFlowQueues: env.RxFlowQueues,
	}
}

func parseVfPairEnv(t testing.TB, tokens []string) (env vfPairEnv) {
	env.t = t
	env.RemoteIfname, env.LocalIfname = tokens[0], tokens[1]
	if localIfTokens := strings.SplitN(env.LocalIfname, "=", 2); len(localIfTokens) == 2 {
		env.LocalIfname = localIfTokens[0]
		pciAddr, _ := pciaddr.Parse(localIfTokens[1])
		env.LocalPCI = &pciAddr
	}
	return
}

type vfTestEnv struct {
	vfPairEnv
	Flags   []string
	RegSel  []string // locator selection in non-flow mode
	FlowSel []string // locator selection in flow mode
}

func parseVfTestEnv(t testing.TB) (env vfTestEnv) {
	line, ok := os.LookupEnv("ETHFACETEST_VF")
	if !ok {
		// ETHFACETEST_VF syntax: remoteIfname,localIfname=localPCI,vfFlags,rxEpsilon,txEpsilon
		// remoteIfname: netif name for the "remote" netif, will be attached with gopacket/afpacket library.
		// localIfname: netif name for the "local" netif, will be attached with DPDK drivers.
		// localPCI (optional): PCI address for the "local" netif; it could be a different VF.
		// vfFlags: "+" separated.
		//   "mcast" enables Ethernet multicast locators.
		//   "vlan" enables VLAN locators.
		//   "gtp" enables GTP-U locators.
		//   "flow" enables RxFlow tests with default selections.
		//   "rss" enables RSS action with 2 queues in VXLAN locators.
		//   "arp-only" restricts pass-through tests to only use ARP traffic.
		// rxEpsilon,txEpsilon: defaults to 0.02.
		//
		// Examples:
		//   ETHFACETEST_VF=enp7s0np0,enp8s0np0,flow
		//   ETHFACETEST_VF=enp4s0f1v0,enp4s0f1v1=04:0a.2,gtp+vlan,0.05,0.05
		t.Skip("VF test disabled; rerun test suite and specify two netifs in ETHFACETEST_VF environ.")
	}

	tokens := strings.Split(line, ",")
	for len(tokens) < 3 {
		tokens = append(tokens, "")
	}
	env.vfPairEnv, env.Flags = parseVfPairEnv(t, tokens), strings.Split(tokens[2], "+")

	if len(tokens) > 4 {
		env.RxLossTolerance, _ = strconv.ParseFloat(tokens[3], 64)
		env.TxLossTolerance, _ = strconv.ParseFloat(tokens[4], 64)
	} else {
		env.RxLossTolerance, env.TxLossTolerance = dfltEpsilon, dfltEpsilon
	}

	exclusions := []string{}
	if !slices.Contains(env.Flags, "vlan") {
		exclusions = append(exclusions, "vlan", "vlan-udp4", "vlan-udp6")
	}
	if !slices.Contains(env.Flags, "mcast") {
		exclusions = append(exclusions, "ether-mcast")
	}
	if len(exclusions) > 0 {
		env.RegSel = append([]string{"-"}, exclusions...)
	}

	env.RxFlowQueues = 4
	if line, ok := os.LookupEnv("ETHFACETEST_VFFLOW"); ok {
		// ETHFACETEST_VFFLOW syntax: number of queues, followed by comma-separated locator titles.
		// If unset but flags contain "flow", 4 queues and default selections are used.
		// This option may be required when "rss" flag is used.
		// Example:
		//   ETHFACETEST_VFFLOW=6,ether,vlan,udp4,udp6
		env.FlowSel = strings.Split(line, ",")
		nQueues, _ := strconv.ParseInt(env.FlowSel[0], 10, 32)
		env.FlowSel = env.FlowSel[1:]
		env.RxFlowQueues = int(nQueues)
	}

	return
}

func TestVfXDP(t *testing.T) {
	env := parseVfTestEnv(t)
	if len(env.RegSel) == 0 {
		env.RegSel = []string{"-", "passthru"}
	} else if env.RegSel[0] == "-" {
		env.RegSel = append(env.RegSel, "passthru")
	}

	prf := env.MakePrf(configPortXDP)
	testPortRemote(prf, env.RegSel)
}

func TestVfAfPacket(t *testing.T) {
	env := parseVfTestEnv(t)
	prf := env.MakePrf(nil)
	testPortRemote(prf, env.RegSel)
}

func TestVfRxTable(t *testing.T) {
	env := parseVfTestEnv(t)
	env.RxFlowQueues = 0
	prf := env.MakePrf(env.ConfigPortPCI)
	testPortRemote(prf, env.RegSel)
}

func TestVfRxFlow(t *testing.T) {
	env := parseVfTestEnv(t)
	if len(env.FlowSel) == 0 {
		if !slices.Contains(env.Flags, "flow") {
			t.Skip("VfRxFlow tests disabled")
		}

		env.FlowSel = []string{"passthru", "ether", "udp4", "udp4p1", "vx4", "udp6", "vx6"}
		if slices.Contains(env.Flags, "gtp") {
			env.FlowSel = append(env.FlowSel, "gtp8", "gtp9")
		}
	}

	modifiers := []string{}
	if slices.Contains(env.Flags, "rss") {
		modifiers = append(modifiers, "+rss")
	}
	if slices.Contains(env.Flags, "arp-only") {
		modifiers = append(modifiers, "+arp-only")
	}

	i := 0
	for group := range slices.Chunk(env.FlowSel, env.RxFlowQueues) {
		group = append(group, modifiers...)
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			env.t = t
			prf := env.MakePrf(env.ConfigPortPCI)
			testPortRemote(prf, group)
		})
		i++
	}
}
