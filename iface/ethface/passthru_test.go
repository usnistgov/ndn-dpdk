package ethface_test

import (
	"bytes"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/afpacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethnetif"
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
			if len(wire) >= 14 && bytes.Equal(wire[6:12], p.Netif.HardwareAddr) {
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

func makePassthru(prf *PortRemoteFixture, loc ethface.PassthruLocator) (p PassthruNetif) {
	p.t = prf.t
	_, require := makeAR(p.t)
	var e error

	loc.EthDev = prf.LocalPort.EthDev()
	p.Face = prf.AddFace(loc)

	p.Netif, e = ethnetif.NetIntfByName(ethport.MakePassthruTapName(prf.LocalPort.EthDev()))
	require.NoError(e)
	require.NoError(p.Netif.EnsureLinkUp())

	return
}

func testPassthru(prf *PortRemoteFixture, arpOnly bool) {
	assert, _ := makeAR(prf.t)

	var locUDP4 ethface.UDPLocator
	locUDP4.Local.HardwareAddr = prf.LocalPort.EthDev().HardwareAddr()
	locUDP4.Remote.HardwareAddr = prf.RemoteMAC
	locUDP4.LocalIP, locUDP4.LocalUDP = netip.MustParseAddr("192.168.2.1"), 6363
	locUDP4.RemoteIP, locUDP4.RemoteUDP = netip.MustParseAddr("192.168.2.2"), 6363
	faceUDP4 := prf.AddFace(locUDP4)

	passthru := makePassthru(prf, ethface.PassthruLocator{})
	passthru.AddIP(netip.PrefixFrom(locUDP4.LocalIP, 24))
	pcapRecv := passthru.EnablePcap()

	prf.DiagFaces()

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
		buf := make([]byte, prf.LocalPort.EthDev().MTU())
		for {
			n, e := prf.RemoteIntf.Read(buf)
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
			prf.RemoteWrite(makeARP(locUDP4.Remote.HardwareAddr, locUDP4.RemoteIP, nil, locUDP4.LocalIP)...)
		case 1, 4, 7: // receive ICMP pings
			prf.RemoteWrite(
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
			iface.TxBurst(faceUDP4.ID(), prf.MakeTxBurst("UDP4", i))
		case 2, 5, 6, 9: // receive NDN packets addressed to faceUDP4
			prf.RemoteWrite(
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
				prf.MakeRxFrame("UDP4", i),
			)
		}
	}
	time.Sleep(10 * time.Millisecond)

	cntUDP4, cntPassthru := faceUDP4.Counters(), passthru.Face.Counters()
	assert.InDelta(48, rxARP.Load(), 8)                       // [40,56]; 50 from case 0 plus kernel generated minus loss
	assert.InDelta(48, txARP.Load(), 8)                       // replies
	assert.InEpsilon(200, cntUDP4.RxInterests, prf.RxEpsilon) // from case 2,5,6,9
	assert.InEpsilon(100, txUDP4.Load(), prf.TxEpsilon)       // from case 3,8
	if arpOnly {
		assert.Zero(rxICMP.Load())                  // ICMP not supported, because passthru face can only receive ARP
		assert.Zero(txICMP.Load())                  // ICMP not supported, because passthru face can only receive ARP
		assert.InDelta(52, cntPassthru.RxFrames, 8) // [44,60]; 50 from case 0 minus loss plus kernel generated
	} else {
		assert.InDelta(145, rxICMP.Load(), 5)         // [140,150]; 150 from case 1,4,7 minus loss
		assert.InDelta(145, txICMP.Load(), 5)         // replies
		assert.InDelta(220, cntPassthru.RxFrames, 20) // [200,240]; 200 from case 0,1,4,7 plus kernel generated
	}
	assert.Less(int(txOther.Load()), 30)
	assert.Greater(int(cntPassthru.RxOctets), 0)
	assert.Greater(int(cntPassthru.TxOctets), 0)
}

func TestPassthruTap(t *testing.T) {
	prf := NewPortRemoteFixture(t, "", "", nil)
	testPassthru(prf, false)
}

func TestPassthruAfPacket(t *testing.T) {
	env := parseVfTestEnv(t)
	prf := env.MakePrf(nil)
	testPassthru(prf, false)
}

func TestPassthruRxTable(t *testing.T) {
	env := parseVfTestEnv(t)
	env.RxFlowQueues = 0
	prf := env.MakePrf(env.ConfigPortPCI)
	testPassthru(prf, false)
}

func TestPassthruRxFlow(t *testing.T) {
	env := parseVfTestEnv(t)
	prf := env.MakePrf(env.ConfigPortPCI)
	testPassthru(prf, strings.Contains(env.Flags, "arp-only"))
}
