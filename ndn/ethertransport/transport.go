//go:build linux

package ethertransport

import (
	"bytes"
	"errors"
	"fmt"
	"net"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/mdlayher/packet"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/ndnlayer"
)

// Config contains Transport configuration.
type Config struct {
	Locator
	MTU int
}

func (cfg *Config) applyDefaults() {
	if cfg.Remote.Empty() {
		cfg.Remote.HardwareAddr = MulticastAddressNDN
	}
}

// Transport is an l3.Transport that communicates over AF_PACKET sockets.
type Transport interface {
	l3.Transport
}

// New creates a Transport.
func New(ifname string, cfg Config) (Transport, error) {
	cfg.applyDefaults()
	if e := cfg.Locator.Validate(); e != nil {
		return nil, fmt.Errorf("cfg.Locator.Validate(): %w", e)
	}

	netif, e := net.InterfaceByName(ifname)
	if e != nil {
		return nil, fmt.Errorf("net.InterfaceByName(%s): %w", ifname, e)
	}

	tr := &transport{}

	tr.ConnHandle, e = NewConnHandle(netif, EthernetTypeNDN)
	if macaddr.IsMulticast(cfg.Remote.HardwareAddr) {
		tr.leave, e = tr.JoinMulticast(cfg.Remote.HardwareAddr)
	} else if !bytes.Equal(cfg.Local.HardwareAddr, netif.HardwareAddr) {
		e = tr.Conn.SetPromiscuous(true)
		tr.leave = func() error { return tr.Conn.SetPromiscuous(false) }
	}
	if e != nil {
		tr.ConnHandle.Close()
		return nil, fmt.Errorf("JoinMulticast/SetPromiscuous: %w", e)
	}

	tr.TransportBase, tr.p = l3.NewTransportBase(l3.TransportBaseConfig{
		MTU: cfg.MTU,
	})

	tr.rx.Prepare(cfg.Locator)
	tr.tx.Prepare(cfg.Locator)
	return tr, nil
}

type transport struct {
	*l3.TransportBase
	p *l3.TransportBasePriv
	*ConnHandle
	leave func() error
	rx    transportRx
	tx    transportTx
}

func (tr *transport) Read(buf []byte) (n int, e error) {
	return tr.rx.Read(tr.Conn, buf)
}

func (tr *transport) Write(buf []byte) (n int, e error) {
	return tr.tx.Write(tr.Conn, buf)
}

func (tr *transport) Close() error {
	defer tr.p.SetState(l3.TransportClosed)
	var errs []error
	if tr.leave != nil {
		errs = append(errs, tr.leave())
	}
	errs = append(errs, tr.ConnHandle.Close())
	return errors.Join(errs...)
}

type transportRx struct {
	expectedSrcAddr net.HardwareAddr
	expectedDstAddr net.HardwareAddr

	parser  *gopacket.DecodingLayerParser
	decoded []gopacket.LayerType
	eth     layers.Ethernet
	dot1q   layers.Dot1Q
	tlv     ndnlayer.TLV
}

func (rx *transportRx) Prepare(loc Locator) {
	if macaddr.IsMulticast(loc.Remote.HardwareAddr) {
		rx.expectedDstAddr = loc.Remote.HardwareAddr
	} else {
		rx.expectedSrcAddr = loc.Remote.HardwareAddr
		rx.expectedDstAddr = loc.Local.HardwareAddr
	}

	rx.parser = gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &rx.eth, &rx.dot1q, &rx.tlv)
	rx.parser.IgnoreUnsupported = true
}

func (rx *transportRx) Read(conn *packet.Conn, buf []byte) (int, error) {
DROP:
	for {
		n, na, e := conn.ReadFrom(buf)
		if e != nil {
			return 0, e
		}
		if rx.expectedSrcAddr != nil && !bytes.Equal(rx.expectedSrcAddr, na.(*packet.Addr).HardwareAddr) {
			continue DROP
		}

		pkt := buf[:n]
		if e = rx.parser.DecodeLayers(pkt, &rx.decoded); e != nil {
			continue
		}

		for _, layerType := range rx.decoded {
			switch layerType {
			case layers.LayerTypeEthernet:
				if !bytes.Equal(rx.expectedDstAddr, rx.eth.DstMAC) {
					continue DROP
				}
			case layers.LayerTypeDot1Q:
				// TODO match VLAN ID
			case ndnlayer.LayerTypeTLV:
				return copy(buf, rx.tlv.LayerContents()), nil
			}
		}
	}
}

type transportTx struct {
	dstAddr packet.Addr

	layers []gopacket.SerializableLayer
	buf    gopacket.SerializeBuffer
	opts   gopacket.SerializeOptions
}

func (tx *transportTx) Prepare(loc Locator) {
	tx.dstAddr.HardwareAddr = loc.Remote.HardwareAddr

	var eth layers.Ethernet
	eth.SrcMAC = loc.Local.HardwareAddr
	eth.DstMAC = loc.Remote.HardwareAddr
	eth.EthernetType = EthernetTypeNDN
	tx.layers = []gopacket.SerializableLayer{&eth}

	if loc.VLAN > 0 {
		var dot1q layers.Dot1Q
		dot1q.Type = eth.EthernetType
		dot1q.VLANIdentifier = uint16(loc.VLAN)
		tx.layers = append(tx.layers, &dot1q)
		eth.EthernetType = layers.EthernetTypeDot1Q
	}

	tx.layers = append(tx.layers, nil)

	tx.buf = gopacket.NewSerializeBuffer()

	tx.opts = gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
}

func (tx *transportTx) Write(conn *packet.Conn, buf []byte) (n int, e error) {
	tx.layers[len(tx.layers)-1] = gopacket.Payload(buf)
	if e := gopacket.SerializeLayers(tx.buf, tx.opts, tx.layers...); e != nil {
		return 0, e
	}
	return conn.WriteTo(tx.buf.Bytes(), &tx.dstAddr)
}
