package ethface_test

import (
	"bytes"
	"net"
	"net/netip"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/afpacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethnetif"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/vishvananda/netlink"
)

// PassthruNetif organizes a pass-through face on a "hardware" TAP netif.
// It also exposes the "inner" TAP netif created by the pass-through face.
type PassthruNetif struct {
	t     testing.TB
	Face  iface.Face
	Netif *ethnetif.NetIntf
	TP    *afpacket.TPacket
}

// AddIP adds an IP address to the "inner" TAP netif.
func (p *PassthruNetif) AddIP(ip netip.Prefix) {
	_, require := makeAR(p.t)

	addr, e := netlink.ParseAddr(ip.String())
	require.NoError(e)
	require.NoError(netlink.AddrAdd(p.Netif.Link, addr))
}

// EnablePcap attaches AF_PACKET socket to the "inner" TAP netif.
func (p *PassthruNetif) EnablePcap() <-chan []byte {
	_, require := makeAR(p.t)
	var e error

	p.TP, e = afpacket.NewTPacket(afpacket.OptInterface(p.Netif.Name))
	require.NoError(e)

	ch := make(chan []byte)
	go func() {
		for {
			wire, _, e := p.TP.ReadPacketData()
			if e != nil {
				close(ch)
				p.TP.Close()
				p.TP = nil
				return
			}
			if len(wire) > 14 && bytes.Equal(wire[6:12], p.Netif.HardwareAddr) {
				// discard loopback packets whose source address is self
				continue
			}
			ch <- wire
		}
	}()
	return ch
}

// SendFromLayers transmits a packet through the AF_PACKET socket created by EnablePcap.
func (p *PassthruNetif) SendFromLayers(hdrs ...gopacket.SerializableLayer) error {
	b, discard := packetFromLayers(hdrs...)
	defer discard()
	return p.TP.WritePacketData(b)
}

func makePassthru(tap *TapFixture, loc ethface.PassthruLocator) (p PassthruNetif) {
	p.t = tap.t
	_, require := makeAR(p.t)
	var e error

	loc.EthDev = tap.Port.EthDev()
	p.Face = tap.AddFace(loc)

	p.Netif, e = ethnetif.NetIntfByName(ethport.MakePassthruTapName(tap.Port.EthDev()))
	require.NoError(e)
	require.NoError(p.Netif.EnsureLinkUp(false))

	return
}

func TestPassthru(t *testing.T) {
	assert, _ := makeAR(t)

	// This TAP netif is the "hardware" EthDev.
	// It is separate from the TAP netif created by the passthru face.
	tap := newTapFixtureAfPacket(t)

	var locUDP4 ethface.UDPLocator
	locUDP4.Local.HardwareAddr = tap.Port.EthDev().HardwareAddr()
	locUDP4.Remote.Set("02:00:00:00:00:02")
	locUDP4.LocalIP, locUDP4.LocalUDP = netip.MustParseAddr("192.168.2.1"), 6363
	locUDP4.RemoteIP, locUDP4.RemoteUDP = netip.MustParseAddr("192.168.2.2"), 6363
	faceUDP4 := tap.AddFace(locUDP4)

	passthru := makePassthru(tap, ethface.PassthruLocator{})
	passthru.AddIP(netip.PrefixFrom(locUDP4.LocalIP, 24))
	pcapRecv := passthru.EnablePcap()

	// "RX" and "TX" are in regards to the "hardware" ethdev.
	// RX packets are received by the "hardware" ethdev; it may go to passthru TAP or faceUDP4.
	// TX packets are sent to the "hardware" ethdev; it may come from passthru TAP or faceUDP4.
	var rxARP, rxICMP, txARP, txICMP, txUDP4, txOther atomic.Int32

	// Count non-NDN packets received on the "inner" TAP netif.
	go func() {
		for pkt := range pcapRecv {
			parsed := gopacket.NewPacket(pkt, layers.LayerTypeEthernet, gopacket.NoCopy)
			if arp, ok := parsed.Layer(layers.LayerTypeARP).(*layers.ARP); ok && arp.Operation == layers.ARPRequest {
				rxARP.Add(1)
				// kernel will send ARP replies
			} else if icmp, ok := parsed.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4); ok && icmp.TypeCode.Type() == layers.ICMPv4TypeEchoRequest {
				rxICMP.Add(1)
				// kernel will send ICMP echo replies
			}
		}
	}()

	// Count packets sent via the "hardware" ethdev.
	go func() {
		buf := make([]byte, tap.Port.EthDev().MTU())
		for {
			n, e := tap.Intf.Read(buf)
			if e != nil {
				break
			}
			pkt := buf[:n]

			parsed := gopacket.NewPacket(pkt, layers.LayerTypeEthernet, gopacket.NoCopy)
			if arp, ok := parsed.Layer(layers.LayerTypeARP).(*layers.ARP); ok && arp.Operation == layers.ARPReply {
				txARP.Add(1)
			} else if icmp, ok := parsed.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4); ok && icmp.TypeCode.Type() == layers.ICMPv4TypeEchoReply {
				txICMP.Add(1)
			} else if udp, ok := parsed.Layer(layers.LayerTypeUDP).(*layers.UDP); ok && udp.DstPort == 6363 {
				txUDP4.Add(1)
			} else {
				txOther.Add(1)
			}
		}
	}()

	for i := range 500 {
		time.Sleep(10 * time.Millisecond)
		switch i % 10 {
		case 0: // receive ARP queries
			tap.WriteToFromLayers(makeARP(locUDP4.Remote.HardwareAddr, locUDP4.RemoteIP, nil, locUDP4.LocalIP)...)
		case 1, 4, 7: // receive ICMP pings
			tap.WriteToFromLayers(
				&layers.Ethernet{
					SrcMAC:       locUDP4.Remote.HardwareAddr,
					DstMAC:       locUDP4.Local.HardwareAddr,
					EthernetType: layers.EthernetTypeIPv4,
				},
				&layers.IPv4{
					Version:  4,
					TTL:      64,
					Protocol: layers.IPProtocolICMPv4,
					SrcIP:    locUDP4.RemoteIP.AsSlice(),
					DstIP:    locUDP4.LocalIP.AsSlice(),
				},
				&layers.ICMPv4{
					TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
					Id:       1,
					Seq:      uint16(i),
				},
			)
		case 3, 8: // send NDN packets from faceUDP4
			iface.TxBurst(faceUDP4.ID(), makeTxBurst("UDP4", i))
		case 2, 5, 6, 9: // receive NDN packets addressed to faceUDP4
			tap.WriteToFromLayers(
				&layers.Ethernet{
					SrcMAC:       locUDP4.Remote.HardwareAddr,
					DstMAC:       locUDP4.Local.HardwareAddr,
					EthernetType: layers.EthernetTypeIPv4,
				},
				&layers.IPv4{
					Version:  4,
					TTL:      64,
					Protocol: layers.IPProtocolUDP,
					SrcIP:    locUDP4.RemoteIP.AsSlice(),
					DstIP:    locUDP4.LocalIP.AsSlice(),
				},
				&layers.UDP{
					SrcPort: layers.UDPPort(locUDP4.RemoteUDP),
					DstPort: layers.UDPPort(locUDP4.LocalUDP),
				},
				makeRxFrame("UDP4", i),
			)
		}
	}
	time.Sleep(10 * time.Millisecond)

	cntUDP4, cntPassthru := faceUDP4.Counters(), passthru.Face.Counters()
	assert.InDelta(48, rxARP.Load(), 8)           // [40,56]; 50 from case 0 plus kernel generated minus loss
	assert.InDelta(48, txARP.Load(), 8)           // replies
	assert.EqualValues(200, cntUDP4.RxInterests)  // from case 2,5,6,9
	assert.EqualValues(100, txUDP4.Load())        // from case 3,8
	assert.InDelta(145, rxICMP.Load(), 5)         // [140,150]; 150 from case 1,4,7 minus loss
	assert.InDelta(145, txICMP.Load(), 5)         // replies
	assert.InDelta(220, cntPassthru.RxFrames, 20) // [200,240]; 200 from case 0,1,4,7 plus kernel generated
	assert.Less(int(txOther.Load()), 30)
	assert.Greater(int(cntPassthru.RxOctets), 0)
	assert.Greater(int(cntPassthru.TxOctets), 0)
}

func addGtpFaces(tap *TapFixture) (faces []iface.Face) {
	local := tap.Port.EthDev().HardwareAddr()
	for i := range 96 {
		var loc ethface.GtpLocator
		loc.Local.HardwareAddr = local
		loc.Remote.HardwareAddr = net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 2 + byte(i>>4)}
		loc.LocalIP = netip.AddrFrom4([4]byte{192, 168, 3, 254})
		loc.RemoteIP = netip.AddrFrom4([4]byte{192, 168, 3, 2 + byte(i>>4)})
		loc.UlTEID, loc.DlTEID = 0x10000000+i, 0x20000000+i
		loc.UlQFI, loc.DlQFI = 2, 12
		loc.InnerLocalIP = netip.AddrFrom4([4]byte{192, 168, 60, 254})
		loc.InnerRemoteIP = netip.AddrFrom4([4]byte{192, 168, 60, 2 + byte(i)})

		faces = append(faces, tap.AddFace(loc))
	}
	return
}

func TestGtpipProcess(t *testing.T) {
	assert, require := makeAR(t)
	tap := newTapFixtureAfPacket(t)
	ethDev := tap.Port.EthDev()

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

	facesGTP := addGtpFaces(tap)
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

func TestGtpipPassthru(t *testing.T) {
	assert, require := makeAR(t)
	tap := newTapFixtureAfPacket(t)
	ethDev := tap.Port.EthDev()
	passthru := makePassthru(tap, ethface.PassthruLocator{
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

	facesGTP := addGtpFaces(tap)
	assert.Equal(len(facesGTP), g.Len())

	var rxICMP, txICMP, txARP atomic.Int32

	// Count non-NDN packets received on the "inner" TAP netif.
	go func() {
		for pkt := range pcapRecv {
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
		buf := make([]byte, tap.Port.EthDev().MTU())
		_, n3net, _ := net.ParseCIDR("192.168.3.0/24")
		for {
			n, e := tap.Intf.Read(buf)
			if e != nil {
				break
			}
			pkt := buf[:n]

			parsed := gopacket.NewPacket(pkt, layers.LayerTypeEthernet, gopacket.NoCopy)
			if arp, ok := parsed.Layer(layers.LayerTypeARP).(*layers.ARP); ok && arp.Operation == layers.ARPRequest {
				if n3net.Contains(arp.DstProtAddress) && arp.DstProtAddress[3] < 0xF0 {
					remoteIP, _ := netip.AddrFromSlice(arp.DstProtAddress)
					remoteMAC := net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, remoteIP.As4()[3]}
					localIP, _ := netip.AddrFromSlice(arp.SourceProtAddress)
					tap.WriteToFromLayers(makeARP(remoteMAC, remoteIP, arp.SourceHwAddress, localIP)...)
				}
				txARP.Add(1)
			} else if icmp, ok := parsed.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4); ok && icmp.TypeCode.Type() == layers.ICMPv4TypeEchoReply {
				txICMP.Add(1)
			}
		}
	}()

	for i := range 120 {
		time.Sleep(10 * time.Millisecond)
		if i < len(facesGTP) {
			tap.WriteToFromLayers(
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
					TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
					Id:       uint16(i),
					Seq:      1,
				},
			)
		} else {
			tap.WriteToFromLayers(
				&layers.Ethernet{
					SrcMAC:       net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 2 + byte(i)},
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
	assert.InDelta(115, rxICMP.Load(), 5)               // [110,120]; 120 pings minus loss
	assert.InDelta(115, txICMP.Load(), 5)               // replies
	assert.InDelta(92, cntPassthru.RxData, 4)           // [88,96]; 96 UE pings minus loss
	assert.InDelta(92, cntPassthru.TxData, 4)           // replies
	assert.InDelta(24+nARP, cntPassthru.RxInterests, 4) // [20,28]; 24 non-UE pings minus loss plus kernel generated
	assert.InDelta(24+nARP, cntPassthru.TxInterests, 4) // replies
	assert.InDelta(30, nARP, 5)                         // [25,35]; 6x N3 peers + 24x non-UE peers, with tolerance
	assert.Greater(int(cntPassthru.RxOctets), 0)
	assert.Greater(int(cntPassthru.TxOctets), 0)
}
