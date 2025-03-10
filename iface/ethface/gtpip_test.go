package ethface_test

import (
	"encoding/hex"
	"net"
	"net/netip"
	"os"
	"slices"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
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
					makeARP(net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0xFE}, netip.AddrFrom4([4]byte{192, 168, 6, 254}),
						nil, netip.AddrFrom4([4]byte{192, 168, 6, 200}))...,
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

func testGtpip(prf *PortRemoteFixture) {
	assert, require := makeAR(prf.t)
	ethDev := prf.LocalPort.EthDev()
	passthru := makePassthru(prf, ethface.PassthruLocator{
		Gtpip: &ethport.GtpipConfig{
			IPv4Capacity: 8192,
		},
	})
	passthru.AddIP(netip.MustParsePrefix("192.168.3.254/24"))
	passthru.AddIP(netip.MustParsePrefix("192.168.6.254/24"))
	require.NoError(netlink.RouteAdd(&netlink.Route{
		LinkIndex: passthru.Netif.Index,
		Dst:       &net.IPNet{IP: net.IPv4(192, 168, 60, 0), Mask: net.CIDRMask(24, 32)},
		Gw:        net.IPv4(192, 168, 3, 200),
	}))
	pcapRecv := passthru.EnablePcap()

	g := ethport.GtpipFromPassthruFace(passthru.Face.ID())
	require.NotNil(g)

	facesGTP := addGtpFaces(prf)
	assert.Equal(len(facesGTP), g.Len())

	prf.DiagFaces()

	dbg, _ := strconv.Atoi(os.Getenv("ETHFACETEST_GTPIPDBG"))
	if dbg >= 1 {
		dbgSleep := func() {
			prf.t.Log("sleep 30 seconds for ETHFACETEST_GTPIPDBG=1")
			time.Sleep(30 * time.Second)
		}
		dbgSleep()
		defer dbgSleep()
	}

	var rxICMP, txICMP, txARP atomic.Int32

	// Count non-NDN packets received on the "inner" TAP netif.
	go func() {
		for pkt := range pcapRecv {
			if dbg >= 2 {
				prf.t.Logf("RX %s", hex.EncodeToString(pkt))
			}

			parsed := gopacket.NewPacket(pkt, layers.LayerTypeEthernet, gopacket.NoCopy)
			if icmp, ok := parsed.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4); ok && icmp.TypeCode.Type() == layers.ICMPv4TypeEchoRequest {
				rxICMP.Add(1)
				// kernel will send ICMP echo replies
			}
		}
	}()

	// Count packets sent via the "hardware" ethdev.
	// Respond to ARP requests.
	go func() {
		buf := make([]byte, prf.LocalPort.EthDev().MTU())
		_, n3net, _ := net.ParseCIDR("192.168.3.0/24")
		for {
			n, e := prf.RemoteIntf.Read(buf)
			if e != nil {
				break
			}
			pkt := buf[:n]

			if dbg >= 2 {
				prf.t.Logf("TX %s", hex.EncodeToString(pkt))
			}

			parsed := gopacket.NewPacket(pkt, layers.LayerTypeEthernet, gopacket.NoCopy)
			if arp, ok := parsed.Layer(layers.LayerTypeARP).(*layers.ARP); ok && arp.Operation == layers.ARPRequest {
				if n3net.Contains(arp.DstProtAddress) && arp.DstProtAddress[3] < 0xF0 {
					remoteIP, _ := netip.AddrFromSlice(arp.DstProtAddress)
					remoteMAC := prf.OverrideMAC(net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, remoteIP.As4()[3]})
					localIP, _ := netip.AddrFromSlice(arp.SourceProtAddress)
					prf.RemoteWrite(makeARP(remoteMAC, remoteIP, arp.SourceHwAddress, localIP)...)
				}
				txARP.Add(1)
			} else if icmp, ok := parsed.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4); ok && icmp.TypeCode.Type() == layers.ICMPv4TypeEchoReply {
				txICMP.Add(1)
			}
		}
	}()

	// Transmit 96 UE pings (one from each UE) and 24 non-UE pings.
	for i := range 120 {
		time.Sleep(10 * time.Millisecond)
		if i < len(facesGTP) {
			prf.RemoteWrite(
				&layers.Ethernet{
					SrcMAC:       prf.OverrideMAC(net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 2 + byte(i>>4)}),
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
					TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
					Id:       uint16(i),
					Seq:      1,
				},
			)
		} else {
			prf.RemoteWrite(
				&layers.Ethernet{
					SrcMAC:       prf.OverrideMAC(net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 2 + byte(i)}),
					DstMAC:       ethDev.HardwareAddr(),
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
	}
	time.Sleep(10 * time.Millisecond)

	nARP, cntPassthru := int(txARP.Load()), passthru.Face.Counters()
	assert.InDelta(112, rxICMP.Load(), 8)               // [104,120]; 120 pings minus loss
	assert.InDelta(112, txICMP.Load(), 8)               // replies
	assert.InDelta(92, cntPassthru.RxData, 4)           // [88,96]; 96 UE pings minus loss
	assert.InDelta(92, cntPassthru.TxData, 4)           // replies
	assert.InDelta(24+nARP, cntPassthru.RxInterests, 6) // [18,30]; 24 non-UE pings minus loss plus kernel generated
	assert.InDelta(24+nARP, cntPassthru.TxInterests, 6) // replies
	assert.InDelta(30, nARP, 10)                        // [20,40]; 6x N3 peers + 24x non-UE peers, with tolerance
	assert.Greater(int(cntPassthru.RxOctets), 0)
	assert.Greater(int(cntPassthru.TxOctets), 0)
}

func TestGtpipTap(t *testing.T) {
	prf := NewPortRemoteFixture(t, "", "", nil)
	prf.RemoteMAC = nil // allow arbitrary remote MAC
	testGtpip(prf)
}

func TestGtpipAfPacket(t *testing.T) {
	env := parseVfTestEnv(t)
	prf := env.MakePrf(nil)
	testGtpip(prf)
}

func TestGtpipRxTable(t *testing.T) {
	env := parseVfTestEnv(t)
	env.RxFlowQueues = 0
	prf := env.MakePrf(env.ConfigPortPCI)
	testGtpip(prf)
}
