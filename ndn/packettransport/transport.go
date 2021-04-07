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
	"go4.org/must"
)

// PacketDataHandle represents a network interface to send and receive Ethernet frames.
type PacketDataHandle interface {
	gopacket.PacketDataSource
	WritePacketData(pkt []byte) error
}

// Config contains Transport configuration.
type Config struct {
	Locator
	l3.TransportQueueConfig
}

func (cfg *Config) applyDefaults() {
	if cfg.Remote.Empty() {
		cfg.Remote.HardwareAddr = MulticastAddressNDN
	}

	cfg.ApplyTransportQueueConfigDefaults()
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
	tr.TransportBase, tr.p = l3.NewTransportBase(cfg.TransportQueueConfig)

	go tr.rxLoop()
	go tr.txLoop()
	return tr, nil
}

type transport struct {
	*l3.TransportBase
	p   *l3.TransportBasePriv
	hdl PacketDataHandle
	loc Locator
}

func (tr *transport) Handle() PacketDataHandle {
	return tr.hdl
}

func (tr *transport) rxLoop() {
	matchSrc := func(a net.HardwareAddr) bool { return macaddr.Equal(tr.loc.Remote.HardwareAddr, a) }
	matchDst := func(a net.HardwareAddr) bool { return macaddr.Equal(tr.loc.Local.HardwareAddr, a) }
	if macaddr.IsMulticast(tr.loc.Remote.HardwareAddr) {
		matchSrc = func(net.HardwareAddr) bool { return true }
		matchDst = func(a net.HardwareAddr) bool { return macaddr.Equal(tr.loc.Remote.HardwareAddr, a) }
	}

	var eth layers.Ethernet
	var payload gopacket.Payload
	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &eth, &payload)
	extractPayload := func(layerType gopacket.LayerType) []byte {
		if layerType == layers.LayerTypeEthernet && eth.EthernetType == EthernetTypeNDN {
			return eth.Payload
		}
		return nil
	}
	if tr.loc.VLAN > 0 {
		var dot1q layers.Dot1Q
		parser.AddDecodingLayer(&dot1q)
		vlan16 := uint16(tr.loc.VLAN)
		extractPayload = func(layerType gopacket.LayerType) []byte {
			if layerType == layers.LayerTypeDot1Q && dot1q.VLANIdentifier == vlan16 && dot1q.Type == EthernetTypeNDN {
				return eth.Payload
			}
			return nil
		}
	}

	decoded := []gopacket.LayerType{}
DROP:
	for {
		packetData, _, e := tr.hdl.ReadPacketData()
		if errors.Is(e, io.EOF) {
			close(tr.p.Rx)
			return
		}
		if e = parser.DecodeLayers(packetData, &decoded); e != nil {
			continue
		}

		for _, layerType := range decoded {
			if layerType == layers.LayerTypeEthernet && (!matchSrc(eth.SrcMAC) || !matchDst(eth.DstMAC)) {
				continue DROP
			}

			payload := extractPayload(layerType)
			if len(payload) == 0 {
				continue
			}

			d := tlv.DecodingBuffer(payload)
			element, e := d.Element()
			if e != nil {
				continue DROP
			}
			select {
			case tr.p.Rx <- element.Wire:
			default:
			}
		}
	}
}

func (tr *transport) txLoop() {
	var eth layers.Ethernet
	eth.SrcMAC = tr.loc.Local.HardwareAddr
	eth.DstMAC = tr.loc.Remote.HardwareAddr
	eth.EthernetType = EthernetTypeNDN
	headers := []gopacket.SerializableLayer{&eth}
	if tr.loc.VLAN > 0 {
		var dot1q layers.Dot1Q
		dot1q.Type = eth.EthernetType
		dot1q.VLANIdentifier = uint16(tr.loc.VLAN)
		headers = append(headers, &dot1q)
		eth.EthernetType = layers.EthernetTypeDot1Q
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	for payload := range tr.p.Tx {
		packetLayers := append([]gopacket.SerializableLayer{}, headers...)
		packetLayers = append(packetLayers, gopacket.Payload(payload))
		if e := gopacket.SerializeLayers(buf, opts, packetLayers...); e != nil {
			continue
		}
		tr.hdl.WritePacketData(buf.Bytes())
	}

	switch hdl := tr.hdl.(type) {
	case io.Closer:
		must.Close(hdl)
	case closer:
		hdl.Close()
	}
	tr.p.SetState(l3.TransportClosed)
}

type closer interface {
	Close()
}

func init() {
	layers.EthernetTypeMetadata[EthernetTypeNDN] = layers.EnumMetadata{
		DecodeWith: gopacket.DecodePayload,
		Name:       "NDN",
		LayerType:  gopacket.LayerTypePayload,
	}
}
