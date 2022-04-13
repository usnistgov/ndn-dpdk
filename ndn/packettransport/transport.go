// Package packettransport implements a transport based on GoPacket library.
// It may be used to create Ethernet faces based on AF_PACKET or libpcap.
package packettransport

import (
	"errors"
	"io"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"golang.org/x/exp/slices"
)

// PacketDataHandle represents a network interface to send and receive Ethernet frames.
type PacketDataHandle interface {
	gopacket.ZeroCopyPacketDataSource
	WritePacketData(pkt []byte) error
}

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

// Transport is an l3.Transport that communicates over a PacketDataHandle.
type Transport interface {
	l3.Transport

	// Handle returns the underlying PacketDataHandle.
	Handle() PacketDataHandle
}

// New creates a Transport.
//
// The transport receives and transmits Ethernet frames via the provided PacketDataHandle.
// If hdl additionally implements io.Closer interface or has `Close()` func that returns nothing,
// it will be invoked when the transport is being closed.
func New(hdl PacketDataHandle, cfg Config) (Transport, error) {
	cfg.applyDefaults()
	if e := cfg.Locator.Validate(); e != nil {
		return nil, e
	}

	tr := &transport{
		hdl: hdl,
		loc: cfg.Locator,
	}
	tr.TransportBase, tr.p = l3.NewTransportBase(l3.TransportBaseConfig{
		MTU: cfg.MTU,
	})

	tr.rx.Prepare(tr.loc)
	tr.tx.Prepare(tr.loc)
	return tr, nil
}

type transport struct {
	*l3.TransportBase
	p   *l3.TransportBasePriv
	hdl PacketDataHandle
	loc Locator
	rx  transportRx
	tx  transportTx
}

func (tr *transport) Handle() PacketDataHandle {
	return tr.hdl
}

func (tr *transport) Read(buf []byte) (n int, e error) {
	return tr.rx.Read(tr.hdl, buf)
}

func (tr *transport) Write(buf []byte) (n int, e error) {
	return tr.tx.Write(tr.hdl, buf)
}

func (tr *transport) Close() error {
	defer tr.p.SetState(l3.TransportClosed)

	type closer interface {
		Close()
	}
	switch hdl := tr.hdl.(type) {
	case io.Closer:
		return hdl.Close()
	case closer:
		hdl.Close()
		return nil
	}
	return nil
}

type transportRx struct {
	eth     layers.Ethernet
	parser  *gopacket.DecodingLayerParser
	decoded []gopacket.LayerType

	matchSrc, matchDst func(a net.HardwareAddr) bool
	extractPayload     func(layerType gopacket.LayerType) []byte
}

func (rx *transportRx) Prepare(loc Locator) {
	rx.matchSrc = func(a net.HardwareAddr) bool { return macaddr.Equal(loc.Remote.HardwareAddr, a) }
	rx.matchDst = func(a net.HardwareAddr) bool { return macaddr.Equal(loc.Local.HardwareAddr, a) }
	if macaddr.IsMulticast(loc.Remote.HardwareAddr) {
		rx.matchSrc = func(net.HardwareAddr) bool { return true }
		rx.matchDst = func(a net.HardwareAddr) bool { return macaddr.Equal(loc.Remote.HardwareAddr, a) }
	}

	var payload gopacket.Payload
	rx.parser = gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &rx.eth, &payload)
	rx.extractPayload = func(layerType gopacket.LayerType) []byte {
		if layerType == layers.LayerTypeEthernet && rx.eth.EthernetType == EthernetTypeNDN {
			return rx.eth.Payload
		}
		return nil
	}
	if loc.VLAN > 0 {
		var dot1q layers.Dot1Q
		rx.parser.AddDecodingLayer(&dot1q)
		vlan16 := uint16(loc.VLAN)
		rx.extractPayload = func(layerType gopacket.LayerType) []byte {
			if layerType == layers.LayerTypeDot1Q && dot1q.VLANIdentifier == vlan16 && dot1q.Type == EthernetTypeNDN {
				return dot1q.Payload
			}
			return nil
		}
	}
}

func (rx *transportRx) Read(hdl PacketDataHandle, buf []byte) (n int, e error) {
DROP:
	for {
		packetData, _, e := hdl.ZeroCopyReadPacketData()
		if errors.Is(e, io.EOF) {
			return 0, e
		}
		if e = rx.parser.DecodeLayers(packetData, &rx.decoded); e != nil {
			continue
		}

		for _, layerType := range rx.decoded {
			if layerType == layers.LayerTypeEthernet && (!rx.matchSrc(rx.eth.SrcMAC) || !rx.matchDst(rx.eth.DstMAC)) {
				continue DROP
			}

			payload := rx.extractPayload(layerType)
			if len(payload) == 0 {
				continue
			}

			d := tlv.DecodingBuffer(payload)
			element, e := d.Element()
			if e != nil {
				continue DROP
			}

			return copy(buf, element.Wire), nil
		}
	}
}

type transportTx struct {
	headers []gopacket.SerializableLayer
	buf     gopacket.SerializeBuffer
	opts    gopacket.SerializeOptions
}

func (tx *transportTx) Prepare(loc Locator) {
	var eth layers.Ethernet
	eth.SrcMAC = loc.Local.HardwareAddr
	eth.DstMAC = loc.Remote.HardwareAddr
	eth.EthernetType = EthernetTypeNDN
	tx.headers = []gopacket.SerializableLayer{&eth}

	if loc.VLAN > 0 {
		var dot1q layers.Dot1Q
		dot1q.Type = eth.EthernetType
		dot1q.VLANIdentifier = uint16(loc.VLAN)
		tx.headers = append(tx.headers, &dot1q)
		eth.EthernetType = layers.EthernetTypeDot1Q
	}

	tx.buf = gopacket.NewSerializeBuffer()

	tx.opts = gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
}

func (tx *transportTx) Write(hdl PacketDataHandle, buf []byte) (n int, e error) {
	packetLayers := append(slices.Clone(tx.headers), gopacket.Payload(buf))
	if e := gopacket.SerializeLayers(tx.buf, tx.opts, packetLayers...); e != nil {
		return 0, e
	}
	return len(buf), hdl.WritePacketData(tx.buf.Bytes())
}

func init() {
	layers.EthernetTypeMetadata[EthernetTypeNDN] = layers.EnumMetadata{
		DecodeWith: gopacket.DecodePayload,
		Name:       "NDN",
		LayerType:  gopacket.LayerTypePayload,
	}
}
