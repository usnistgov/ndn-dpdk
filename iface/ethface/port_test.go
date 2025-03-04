package ethface_test

import (
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
	goafpacket "github.com/gopacket/gopacket/afpacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/songgao/water"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/core/pciaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethnetif"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/ndnlayer"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport/afpacket"
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
	t          testing.TB
	RemoteMAC  net.HardwareAddr
	RemoteIntf io.ReadWriteCloser
	LocalPort  *ethport.Port
	RxEpsilon  float64
	TxEpsilon  float64
}

// AddFace creates a face in the local port.
func (prf *PortRemoteFixture) AddFace(loc ethport.Locator) iface.Face {
	_, require := makeAR(prf.t)
	face, e := loc.CreateFace()
	require.NoError(e)
	prf.t.Cleanup(func() { must.Close(face) })
	return face
}

// RemoteWrite writes an Ethernet frame via the remote intf.
// This typically causes the local port to receive the frame.
func (prf *PortRemoteFixture) RemoteWrite(hdrs ...gopacket.SerializableLayer) {
	assert, _ := makeAR(prf.t)
	_, e := writeToFromLayers(prf.RemoteIntf, hdrs...)
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
	prf := &PortRemoteFixture{
		t:         t,
		RxEpsilon: dfltEpsilon,
		TxEpsilon: dfltEpsilon,
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
		e = link.EnsureLinkUp(false)
		require.NoError(e)
		prf.RemoteMAC = link.HardwareAddr

		tpacket, e := goafpacket.NewTPacket(goafpacket.OptInterface(remoteIfname), goafpacket.OptAddVLANHeader(true))
		require.NoError(e)
		prf.RemoteIntf = afpacket.NewTPacketHandle(tpacket)
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
	t.Cleanup(func() { must.Close(prf.LocalPort) })

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

func testPortRemote(prf *PortRemoteFixture, selections string) {
	assert, _ := makeAR(prf.t)

	var disabled, enabled []string
	if strings.HasPrefix(selections, "-") {
		disabled = strings.Split(selections[1:], ",")
	} else if selections != "" {
		enabled = strings.Split(selections, ",")
	}
	type portRemoteFaceRecord struct {
		Title string
		Face  iface.Face
		TxCnt atomic.Int32
	}
	faces := map[string]*portRemoteFaceRecord{}
	addFaceIfEnabled := func(title string, loc ethport.Locator) {
		if (len(disabled) > 0 && slices.Contains(disabled, title)) || (len(enabled) > 0 && !slices.Contains(enabled, title)) {
			return
		}
		faces[title] = &portRemoteFaceRecord{
			Face: prf.AddFace(loc),
		}
	}

	var locEther ethface.EtherLocator
	locEther.Local.HardwareAddr = prf.LocalPort.EthDev().HardwareAddr()
	locEther.Remote.HardwareAddr = prf.RemoteMAC
	addFaceIfEnabled("ether", locEther)

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

	var txOther atomic.Int32
	go func() {
		buf := make([]byte, prf.LocalPort.EthDev().MTU())
		for {
			n, e := prf.RemoteIntf.Read(buf)
			if e != nil {
				break
			}

			classify, isV4, isVlan, gtp := "", false, false, 0
			parsed := gopacket.NewPacket(buf[:n], layers.LayerTypeEthernet, gopacket.NoCopy)
			for i, l := range parsed.Layers() {
				switch l := l.(type) {
				case *layers.Ethernet:
					if i == 0 && l.EthernetType == an.EtherTypeNDN {
						classify = "ether"
					}
				case *layers.Dot1Q:
					isVlan = true
					if l.Type == an.EtherTypeNDN {
						classify = "vlan"
					}
				case *layers.IPv4:
					isV4 = true
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

	for i := range 500 {
		time.Sleep(10 * time.Millisecond)

		if loc, rec := locEther, faces["ether"]; rec != nil {
			prf.RemoteWrite(
				&layers.Ethernet{SrcMAC: loc.Remote.HardwareAddr, DstMAC: loc.Local.HardwareAddr, EthernetType: an.EtherTypeNDN},
				prf.MakeRxFrame("ether", i),
			)
			iface.TxBurst(rec.Face.ID(), prf.MakeTxBurst("ether", i))
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
		assert.InEpsilon(500, rec.Face.Counters().RxInterests, prf.RxEpsilon, title)
		assert.InEpsilon(500, rec.TxCnt.Load(), prf.TxEpsilon, title)
	}
	assert.Less(int(txOther.Load()), 50)
}

func TestTapXDP(t *testing.T) {
	prf := NewPortRemoteFixture(t, "", "", configPortXDP)
	testPortRemote(prf, "")
}

func TestTapAfPacket(t *testing.T) {
	prf := NewPortRemoteFixture(t, "", "", nil)
	testPortRemote(prf, "tapSel")
}

type vfTestEnv struct {
	t            testing.TB
	RemoteIfname string
	LocalIfname  string
	LocalPCI     *pciaddr.PCIAddress
	Flags        string
	RxEpsilon    float64
	TxEpsilon    float64

	RegSel       string   // locator selection in non-flow mode
	FlowSel      []string // locator selection in flow mode
	RxFlowQueues int
}

func (env *vfTestEnv) MakePrf(configPort func(ifname string) ethport.Config) *PortRemoteFixture {
	prf := NewPortRemoteFixture(env.t, env.RemoteIfname, env.LocalIfname, configPort)
	prf.RxEpsilon, prf.TxEpsilon = env.RxEpsilon, env.TxEpsilon
	return prf
}

func (env *vfTestEnv) ConfigPortPCI(ifname string) ethport.Config {
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

func parseVfTestEnv(t *testing.T) (env vfTestEnv) {
	env.t = t

	line, ok := os.LookupEnv("ETHFACETEST_VF")
	if !ok {
		// ETHFACETEST_VF syntax: remoteIfname,localIfname=localPCI,vfFlags,rxEpsilon,txEpsilon
		// remoteIfname: netif name for the "remote" netif, will be attached with gopacket/afpacket library.
		// localIfname: netif name for the "local" netif, will be attached with DPDK drivers.
		// localPCI (optional): PCI address for the "local" netif; it could be a different VF.
		// vfFlags: "+" separated.
		//   "vlan" enables VLAN locators.
		//   "gtp" enables GTP-U locators.
		//   "flow" enables RxFlow tests with default selections.
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
	env.RemoteIfname, env.LocalIfname, env.Flags = tokens[0], tokens[1], tokens[2]

	if localIfTokens := strings.SplitN(env.LocalIfname, "=", 2); len(localIfTokens) == 2 {
		env.LocalIfname = localIfTokens[0]
		pciAddr, _ := pciaddr.Parse(localIfTokens[1])
		env.LocalPCI = &pciAddr
	}

	if len(tokens) > 4 {
		env.RxEpsilon, _ = strconv.ParseFloat(tokens[3], 64)
		env.TxEpsilon, _ = strconv.ParseFloat(tokens[4], 64)
	} else {
		env.RxEpsilon, env.TxEpsilon = dfltEpsilon, dfltEpsilon
	}

	env.RegSel = "-vlan,vlan-udp4,vlan-udp6"
	if strings.Contains(env.Flags, "vlan") {
		env.RegSel = ""
	}

	env.RxFlowQueues = 4
	if line, ok := os.LookupEnv("ETHFACETEST_VFFLOW"); ok {
		// ETHFACETEST_VFFLOW syntax: number of queues, followed by comma-separated locator titles.
		// If unset but flags contain "flow", 4 queues and default selections are used.
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
		if !strings.Contains(env.Flags, "flow") {
			t.Skip("VfRxFlow tests disabled")
		}

		env.FlowSel = []string{"ether", "udp4", "udp4p1", "udp6", "vx4", "vx6"}
		if strings.Contains(env.Flags, "gtp") {
			env.FlowSel = append(env.FlowSel, "gtp8", "gtp9")
		}
	}

	i := 0
	for group := range slices.Chunk(env.FlowSel, env.RxFlowQueues) {
		sel := strings.Join(group, ",")
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			env.t = t
			prf := env.MakePrf(env.ConfigPortPCI)
			testPortRemote(prf, sel)
		})
		i++
	}
}
