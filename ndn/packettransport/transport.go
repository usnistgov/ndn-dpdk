// Package packettransport implements a transport based on gopacket library.
// It may be used to create Ethernet faces based on AF_PACKET or tcpdump.
package packettransport

import (
	"errors"
	"io"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/usnistgov/ndn-dpdk/core/emission"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// PacketDataHandle represents a network interface to send and receive Ethernet frames.
type PacketDataHandle interface {
	gopacket.PacketDataSource
	WritePacketData(pkt []byte) error
}

// Config contains Transport configuration.
type Config struct {
	Locator

	// RxChanBuffer is the Go channel buffer size of RX channel.
	// The default is 64.
	RxQueueSize int

	// TxChanBuffer is the Go channel buffer size of TX channel.
	// The default is 64.
	TxQueueSize int
}

func (cfg *Config) applyDefaults() {
	if len(cfg.Remote) == 0 {
		cfg.Remote = MulticastAddressNDN
	}
	if cfg.RxQueueSize <= 0 {
		cfg.RxQueueSize = 64
	}
	if cfg.TxQueueSize <= 0 {
		cfg.TxQueueSize = 64
	}
}

// Transport is an ndn.Transport that communicates over a PacketDataHandle.
type Transport interface {
	ndn.Transport

	// Handle returns the underlying PacketDataHandle.
	Handle() PacketDataHandle

	// OnClose registers a callback to be invoked when the transport is closed.
	OnClose(cb func()) io.Closer
}

// New creates a Transport.
func New(hdl PacketDataHandle, cfg Config) (Transport, error) {
	cfg.applyDefaults()
	if e := cfg.Locator.Validate(); e != nil {
		return nil, e
	}

	tr := &transport{
		hdl:     hdl,
		loc:     cfg.Locator,
		emitter: emission.NewEmitter(),
		rx:      make(chan []byte, cfg.RxQueueSize),
		tx:      make(chan []byte, cfg.TxQueueSize),
	}
	go tr.rxLoop()
	go tr.txLoop()
	return tr, nil
}

type transport struct {
	hdl     PacketDataHandle
	loc     Locator
	emitter *emission.Emitter
	rx      chan []byte
	tx      chan []byte
}

func (tr *transport) Rx() <-chan []byte {
	return tr.rx
}

func (tr *transport) Tx() chan<- []byte {
	return tr.tx
}

func (tr *transport) Handle() PacketDataHandle {
	return tr.hdl
}

func (tr *transport) OnClose(cb func()) io.Closer {
	return tr.emitter.On(eventClose, cb)
}

func (tr *transport) rxLoop() {
	matchSrc := func(a net.HardwareAddr) bool { return macaddr.Equal(tr.loc.Remote, a) }
	matchDst := func(a net.HardwareAddr) bool { return macaddr.Equal(tr.loc.Local, a) }
	if macaddr.IsMulticast(tr.loc.Remote) {
		matchSrc = func(net.HardwareAddr) bool { return true }
		matchDst = func(a net.HardwareAddr) bool { return macaddr.Equal(tr.loc.Remote, a) }
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
			close(tr.rx)
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

			d := tlv.Decoder(payload)
			element, e := d.Element()
			if e != nil {
				continue DROP
			}
			select {
			case tr.rx <- element.Wire:
			default:
			}
		}
	}
}

func (tr *transport) txLoop() {
	var eth layers.Ethernet
	eth.SrcMAC = tr.loc.Local
	eth.DstMAC = tr.loc.Remote
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
	for payload := range tr.tx {
		packetLayers := append([]gopacket.SerializableLayer{}, headers...)
		packetLayers = append(packetLayers, gopacket.Payload(payload))
		if e := gopacket.SerializeLayers(buf, opts, packetLayers...); e != nil {
			continue
		}
		tr.hdl.WritePacketData(buf.Bytes())
	}

	switch hdl := tr.hdl.(type) {
	case io.Closer:
		hdl.Close()
	case closer:
		hdl.Close()
	}
}

type closer interface {
	Close()
}

const (
	eventClose = "Close"
)

func init() {
	layers.EthernetTypeMetadata[EthernetTypeNDN] = layers.EnumMetadata{
		DecodeWith: gopacket.DecodePayload,
		Name:       "NDN",
		LayerType:  gopacket.LayerTypePayload,
	}
}
