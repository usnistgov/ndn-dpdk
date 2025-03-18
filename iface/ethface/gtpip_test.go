package ethface_test

import (
	"bytes"
	"encoding/hex"
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
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport/afpacket"
	"github.com/vishvananda/netlink"
)

// addGtpFaces creates 96 GTP-U faces.
func addGtpFaces(prf *PortRemoteFixture) (faces []iface.Face) {
	local := prf.LocalPort.EthDev().HardwareAddr()
	for i := range 96 {
		var loc ethface.GtpLocator
		loc.Local.HardwareAddr = local
		loc.Remote.HardwareAddr = prf.OverrideMAC(net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 2 + byte(i>>4)})
		loc.LocalIP = netip.AddrFrom4([4]byte{192, 168, 3, 254})
		loc.RemoteIP = netip.AddrFrom4([4]byte{192, 168, 3, 2 + byte(i>>4)})
		loc.UlTEID, loc.DlTEID = 0x10000000+i, 0x20000000+i
		loc.UlQFI, loc.DlQFI = 2, 12
		loc.InnerLocalIP = netip.AddrFrom4([4]byte{192, 168, 60, 254})
		loc.InnerRemoteIP = netip.AddrFrom4([4]byte{192, 168, 60, 2 + byte(i)})

		faces = append(faces, prf.AddFace(loc))
	}
	return
}

func TestGtpipProcess(t *testing.T) {
	assert, require := makeAR(t)
	prf := NewPortRemoteFixture(t, "", "", nil)
	ethDev := prf.LocalPort.EthDev()

	g, e := ethport.NewGtpip(ethport.GtpipConfig{
		IPv4Capacity: 8192,
	}, ethDev.NumaSocket())
	require.NoError(e)

	process100Pkts := func(f func(vec pktmbuf.Vector) []bool, vec pktmbuf.Vector) []bool {
		require.Len(vec, 100)
		// put non-matching packets in the middle
		matches := f(slices.Concat(vec[:50], vec[70:], vec[50:70]))
		return slices.Concat(matches[:50], matches[80:], matches[50:80])
	}

	prf.RemoteMAC = nil
	facesGTP := addGtpFaces(prf)
	for i, face := range facesGTP {
		e = g.Insert(face.Locator().(ethface.GtpLocator).InnerRemoteIP, face)
		assert.NoError(e, "%d", i)
	}
	assert.Equal(len(facesGTP), g.Len())

	t.Run("ProcessDownlink", func(t *testing.T) {
		assert, _ := makeAR(t)
		vec := make(pktmbuf.Vector, 100)
		defer vec.Close()
		pktLens := make([]int, len(vec))
		for i := range vec {
			switch i {
			case 99:
				vec[i] = pktmbufFromLayers(pktmbuf.DefaultHeadroom,
					makeARP(net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0xFE}, net.IPv4(192, 168, 6, 254), nil, net.IPv4(192, 168, 6, 200))...,
				)
			default:
				vec[i] = pktmbufFromLayers(pktmbuf.DefaultHeadroom,
					&layers.Ethernet{
						SrcMAC:       net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0xFE},
						DstMAC:       ethDev.HardwareAddr(),
						EthernetType: layers.EthernetTypeIPv4,
					},
					&layers.IPv4{
						Version:  4,
						TTL:      64,
						Protocol: layers.IPProtocolICMPv4,
						SrcIP:    net.IPv4(192, 168, 6, 254),
						DstIP:    net.IPv4(192, 168, 60, 2+byte(i)),
					},
					&layers.ICMPv4{
						TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
						Id:       uint16(i),
						Seq:      1,
					},
				)
			}
			pktLens[i] = vec[i].Len()
		}

		matches := process100Pkts(g.ProcessDownlink, vec)
		for i, pkt := range vec {
			if i >= len(facesGTP) {
				assert.False(matches[i], "%d", i)
				assert.Equal(pktLens[i], pkt.Len(), "%d", i)
			} else if assert.True(matches[i], "%d", i) {
				loc := facesGTP[i].Locator().(ethface.GtpLocator)
				wire := pkt.Bytes()
				parsed := checkPacketLayers(t, wire,
					layers.LayerTypeEthernet, layers.LayerTypeIPv4, layers.LayerTypeUDP, layers.LayerTypeGTPv1U,
					layers.LayerTypeIPv4, layers.LayerTypeICMPv4)
				gtp := parsed.Layer(layers.LayerTypeGTPv1U).(*layers.GTPv1U)
				assert.EqualValues(loc.DlTEID, gtp.TEID, "%d", i)
			}
		}
		vec.Close()
	})

	t.Run("ProcessUplink", func(t *testing.T) {
		assert, _ := makeAR(t)
		vec := make(pktmbuf.Vector, 100)
		defer vec.Close()
		pktLens := make([]int, len(vec))
		for i := range vec {
			vec[i] = pktmbufFromLayers(pktmbuf.DefaultHeadroom,
				&layers.Ethernet{
					SrcMAC:       net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 2 + byte(i>>4)},
					DstMAC:       ethDev.HardwareAddr(),
					EthernetType: layers.EthernetTypeIPv4,
				},
				&layers.IPv4{
					Version:  4,
					TTL:      64,
					Protocol: layers.IPProtocolUDP,
					SrcIP:    net.IPv4(192, 168, 3, 2+byte(i>>4)),
					DstIP:    net.IPv4(192, 168, 3, 254),
				},
				&layers.UDP{
					SrcPort: ethport.UDPPortGTP,
					DstPort: ethport.UDPPortGTP,
				},
				makeGTPv1U(0x10000000+uint32(i), 1, 2),
				&layers.IPv4{
					Version:  4,
					TTL:      64,
					Protocol: layers.IPProtocolICMPv4,
					SrcIP:    net.IPv4(192, 168, 60, 2+byte(i)),
					DstIP:    net.IPv4(192, 168, 6, 254),
				},
				&layers.ICMPv4{
					TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoReply, 0),
					Id:       uint16(i),
					Seq:      1,
				},
			)
			pktLens[i] = vec[i].Len()
		}

		matches := process100Pkts(g.ProcessUplink, vec)
		for i, pkt := range vec {
			if i >= len(facesGTP) {
				assert.False(matches[i], "%d", i)
				assert.Equal(pktLens[i], pkt.Len(), "%d", i)
			} else if assert.True(matches[i], "%d", i) {
				wire := pkt.Bytes()
				checkPacketLayers(t, wire,
					layers.LayerTypeEthernet, layers.LayerTypeIPv4, layers.LayerTypeICMPv4)
			}
		}
	})
}

type GtpipFixture struct {
	DebugLevel int

	*PortRemoteFixture
	PassthruFace iface.Face
	Gtpip        *ethport.Gtpip

	N6     *PortRemoteFixture // if specified, create and use N6 face
	N6Face iface.Face

	GtpFaces []iface.Face
}

func (gf *GtpipFixture) Setup() {
	assert, require := makeAR(gf.t)
	gf.DebugLevel, _ = strconv.Atoi(os.Getenv("ETHFACETEST_GTPIPDBG"))

	if gf.N6 != nil {
		gf.N6Face = gf.N6.AddFace(ethface.PassthruLocator{
			FaceConfig: ethport.FaceConfig{
				EthDev: gf.N6.LocalPort.EthDev(),
			},
		})
	}

	ptLoc := ethface.PassthruLocator{
		FaceConfig: ethport.FaceConfig{
			EthDev: gf.LocalPort.EthDev(),
		},
		Gtpip: &ethport.GtpipConfig{
			IPv4Capacity: 8192,
		},
	}
	if gf.N6Face != nil {
		ptLoc.Gtpip.N6Face = gf.N6Face
		ptLoc.Gtpip.N6Local.HardwareAddr = gf.N6.LocalPort.EthDev().HardwareAddr()
		ptLoc.Gtpip.N6Remote.HardwareAddr = gf.N6.RemoteMAC
	}
	gf.PassthruFace = gf.AddFace(ptLoc)
	gf.Gtpip = ethport.GtpipFromPassthruFace(gf.PassthruFace.ID())
	require.NotNil(gf.Gtpip)

	if gf.N6 == nil {
		gf.SetupPassthruNetif(
			netip.MustParsePrefix("192.168.3.254/24"),
			netip.MustParsePrefix("192.168.6.254/24"),
		)
		require.NoError(netlink.RouteReplace(&netlink.Route{
			LinkIndex: gf.PassthruNetif.Index,
			Dst:       &net.IPNet{IP: net.IPv4(192, 168, 60, 0), Mask: net.CIDRMask(24, 32)},
			Gw:        net.IPv4(192, 168, 3, 200),
		}))
	} else {
		gf.SetupPassthruNetif(netip.MustParsePrefix("192.168.3.254/24"))
		gf.N6.SetupPassthruNetif()
		require.NoError(gf.N6.RemoteNetif.SetIP(
			netip.MustParsePrefix("192.168.6.254/24"),
		))
		_, ueNet, _ := net.ParseCIDR("192.168.60.0/24")
		require.NoError(netlink.RouteReplace(&netlink.Route{
			LinkIndex: gf.N6.RemoteNetif.Index,
			Dst:       ueNet,
			Gw:        net.IPv4(192, 168, 6, 200),
		}))
	}

	gf.GtpFaces = addGtpFaces(gf.PortRemoteFixture)
	assert.Equal(len(gf.GtpFaces), gf.Gtpip.Len())
}

// N3Ping causes GTP-IP to receive an ICMPv4 packet from N3 peer.
// If i<len(GtpFaces), the packet comes from an UE whose IP address is derived from i.
// Otherwise, the packet comes from a non-UE peer whose IP address is derived from i.
func (gf *GtpipFixture) N3Ping(i int) {
	if i >= len(gf.GtpFaces) {
		gf.n3PingPlain(i)
	} else {
		gf.n3PingGtp(i)
	}
}

func (gf *GtpipFixture) n3PingPlain(i int) {
	gf.RemoteWrite(
		&layers.Ethernet{
			SrcMAC:       gf.OverrideMAC(net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 2 + byte(i)}),
			DstMAC:       gf.LocalPort.EthDev().HardwareAddr(),
			EthernetType: layers.EthernetTypeIPv4,
		},
		&layers.IPv4{
			Version:  4,
			TTL:      64,
			Protocol: layers.IPProtocolICMPv4,
			SrcIP:    net.IPv4(192, 168, 3, 2+byte(i)),
			DstIP:    net.IPv4(192, 168, 3, 254),
		},
		&layers.ICMPv4{
			TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
			Id:       uint16(i),
			Seq:      1,
		},
	)
}

func (gf *GtpipFixture) n3PingGtp(i int) {
	gf.RemoteWrite(
		&layers.Ethernet{
			SrcMAC:       gf.OverrideMAC(net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 2 + byte(i>>4)}),
			DstMAC:       gf.LocalPort.EthDev().HardwareAddr(),
			EthernetType: layers.EthernetTypeIPv4,
		},
		&layers.IPv4{
			Version:  4,
			TTL:      64,
			Protocol: layers.IPProtocolUDP,
			SrcIP:    net.IPv4(192, 168, 3, 2+byte(i>>4)),
			DstIP:    net.IPv4(192, 168, 3, 254),
		},
		&layers.UDP{
			SrcPort: ethport.UDPPortGTP,
			DstPort: ethport.UDPPortGTP,
		},
		makeGTPv1U(0x10000000+uint32(i), 1, 2),
		&layers.IPv4{
			Version:  4,
			TTL:      64,
			Protocol: layers.IPProtocolICMPv4,
			SrcIP:    net.IPv4(192, 168, 60, 2+byte(i)),
			DstIP:    net.IPv4(192, 168, 6, 254),
		},
		&layers.ICMPv4{
			TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
			Id:       uint16(i),
			Seq:      1,
		},
	)
}

// CountRX counts ICMP echo requests.
func (gf *GtpipFixture) CountRX(intf io.Reader, selfMAC net.HardwareAddr, rxICMP *atomic.Int32) {
	buf := make([]byte, 9200)
	for {
		n, e := intf.Read(buf)
		if e != nil {
			break
		}
		if n < 14 || bytes.Equal(buf[6:12], selfMAC) {
			continue
		}
		pkt := buf[:n]

		if gf.DebugLevel >= 2 {
			gf.t.Logf("RX %s", hex.EncodeToString(pkt))
		}

		parsed := gopacket.NewPacket(pkt, layers.LayerTypeEthernet, gopacket.NoCopy)
		if icmp, ok := parsed.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4); ok && icmp.TypeCode.Type() == layers.ICMPv4TypeEchoRequest {
			rxICMP.Add(1)
			// kernel will send ICMP echo replies
		}
	}
}

// CountTX counts ICMP echo replies and responds to ARP requests.
func (gf *GtpipFixture) CountTX(
	intf io.ReadWriter, selfMAC net.HardwareAddr, txICMP *atomic.Int32,
	replyARP func(ip net.IP) bool,
) {
	buf := make([]byte, 9200)
	for {
		n, e := gf.RemoteIntf.Read(buf)
		if e != nil {
			break
		}
		if n < 14 || bytes.Equal(buf[6:12], selfMAC) {
			continue
		}
		pkt := buf[:n]

		if gf.DebugLevel >= 2 {
			gf.t.Logf("TX %s", hex.EncodeToString(pkt))
		}

		parsed := gopacket.NewPacket(pkt, layers.LayerTypeEthernet, gopacket.NoCopy)
		if arp, ok := parsed.Layer(layers.LayerTypeARP).(*layers.ARP); ok && arp.Operation == layers.ARPRequest {
			if replyARP(arp.DstProtAddress) {
				remoteMAC := append(net.HardwareAddr{0x02, 0x00}, arp.DstProtAddress...)
				gf.RemoteWrite(
					makeARP(gf.OverrideMAC(remoteMAC), arp.DstProtAddress, arp.SourceHwAddress, arp.SourceProtAddress)...,
				)
			}
		} else if icmp, ok := parsed.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4); ok && icmp.TypeCode.Type() == layers.ICMPv4TypeEchoReply {
			txICMP.Add(1)
		}
	}
}

func testGtpip(gf *GtpipFixture) {
	assert, require := makeAR(gf.t)
	gf.Setup()

	gf.DiagFaces()

	if gf.DebugLevel >= 1 {
		dbgSleep := func() {
			gf.t.Log("sleep 30 seconds for ETHFACETEST_GTPIPDBG=1")
			time.Sleep(30 * time.Second)
		}
		dbgSleep()
		defer dbgSleep()
	}

	var rxICMP, txICMP atomic.Int32

	// Count non-NDN packets received on the "inner" TAP netif.
	{
		tpacket, e := goafpacket.NewTPacket(goafpacket.OptInterface(gf.PassthruNetif.Name), goafpacket.OptAddVLANHeader(true))
		require.NoError(e)
		passthruIntf := afpacket.NewTPacketHandle(tpacket)
		// TPacketHandle closes itself when the TAP netif goes away.
		go gf.CountRX(passthruIntf, gf.PassthruNetif.HardwareAddr, &rxICMP)
	}

	// Count packets sent via the "hardware" ethdev.
	// Respond to ARP requests.
	_, n3net, _ := net.ParseCIDR("192.168.3.0/24")
	go gf.CountTX(gf.RemoteIntf, gf.RemoteMAC, &txICMP, func(ip net.IP) bool {
		return n3net.Contains(ip) && ip[3] < 0xF0
	})

	if gf.N6 != nil {
		go gf.CountRX(gf.N6.RemoteIntf, gf.N6.RemoteMAC, &rxICMP)

		tpacket, e := goafpacket.NewTPacket(goafpacket.OptInterface(gf.N6.PassthruNetif.Name), goafpacket.OptAddVLANHeader(true))
		require.NoError(e)
		passthruIntf := afpacket.NewTPacketHandle(tpacket)
		// TPacketHandle closes itself when the TAP netif goes away.
		go gf.CountTX(passthruIntf, gf.N6.PassthruNetif.HardwareAddr, &txICMP, func(ip net.IP) bool {
			return ip.Equal(net.IPv4(192, 168, 6, 200))
		})
	}

	// Transmit 96 UE pings (one from each UE) and 24 non-UE pings.
	for i := range 120 {
		time.Sleep(10 * time.Millisecond)
		gf.N3Ping(i)
	}
	time.Sleep(10 * time.Millisecond)

	cntPassthru := gf.PassthruFace.Counters()
	assert.InDelta(112, rxICMP.Load(), 8)            // [104,120]; 120 pings minus loss
	assert.InDelta(112, txICMP.Load(), 8)            // replies
	assert.InDelta(92, cntPassthru.RxData, 4)        // [88,96]; 96 UE pings minus loss
	assert.Greater(int(cntPassthru.RxInterests), 24) // 24 non-UE pings plus ARP
	if gf.N6 == nil {
		assert.InDelta(92, cntPassthru.TxData, 4)        // replies to UE pings
		assert.Greater(int(cntPassthru.TxInterests), 24) // replies to non-UE pings plus ARP
	} else {
		assert.Zero(cntPassthru.TxData)
		assert.Greater(int(cntPassthru.TxInterests), 120) // replies to all pings
	}
	assert.Greater(int(cntPassthru.RxOctets), 0)
	assert.Greater(int(cntPassthru.TxOctets), 0)

	if gf.N6Face != nil {
		cntN6 := gf.N6Face.Counters()
		gf.t.Log("N6 counters", cntN6)
	}
}

func TestGtpipTap(t *testing.T) {
	prf := NewPortRemoteFixture(t, "", "", nil)
	prf.RemoteMAC = nil // allow arbitrary remote MAC
	testGtpip(&GtpipFixture{PortRemoteFixture: prf})
}

func TestGtpipAfPacket(t *testing.T) {
	env := parseVfTestEnv(t)
	testGtpip(&GtpipFixture{PortRemoteFixture: env.MakePrf(nil)})
}

func TestGtpipRxTable(t *testing.T) {
	env := parseVfTestEnv(t)
	env.RxFlowQueues = 0
	testGtpip(&GtpipFixture{PortRemoteFixture: env.MakePrf(env.ConfigPortPCI)})
}

func TestGtpipN6(t *testing.T) {
	envN3 := parseVfTestEnv(t)
	envN3.RxFlowQueues = 0

	line, ok := os.LookupEnv("ETHFACETEST_VFN6")
	if !ok {
		// ETHFACETEST_VFN6 syntax: remoteIfname,localIfname=localPCI
		t.Skip("GTPIP-N6 test disabled; rerun test suite and specify two netifs in ETHFACETEST_VFN6 environ.")
	}
	envN6 := parseVfPairEnv(t, strings.Split(line, ","))
	envN6.RxFlowQueues = 0

	gf := GtpipFixture{
		PortRemoteFixture: envN3.MakePrf(envN3.ConfigPortPCI),
		N6:                envN6.MakePrf(envN6.ConfigPortPCI),
	}
	testGtpip(&gf)
}
