// Package packettransport implements a transport based on GoPacket library.
// It may be used to create Ethernet faces based on AF_PACKET or libpcap.
package packettransport

import (
	"errors"
	"io"
	"net"

	"slices"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/ndnlayer"
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

// Transport is an l3.Transport that communicates over Ethernet via PacketDataHandle.
type Transport interface {
	l3.Transport

	// Handle returns the underlying PacketDataHandle.
	Handle() PacketDataHandle
}

// New creates a Transport.
//
// The transport receives and transmits Ethernet frames via the provided PacketDataHandle.
// If hdl has either `Close() error` or `Close()` method, it is invoked when transport is being closed.
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
	parser  *gopacket.DecodingLayerParser
	decoded []gopacket.LayerType
	eth     layers.Ethernet
	dot1q   layers.Dot1Q
	tlv     ndnlayer.TLV

	matchLL func(src, dst net.HardwareAddr) bool
}

func (rx *transportRx) Prepare(loc Locator) {
	if macaddr.IsMulticast(loc.Remote.HardwareAddr) {
		rx.matchLL = func(src, dst net.HardwareAddr) bool {
			return macaddr.Equal(loc.Remote.HardwareAddr, dst)
		}
	} else {
		rx.matchLL = func(src, dst net.HardwareAddr) bool {
			return macaddr.Equal(loc.Remote.HardwareAddr, src) && macaddr.Equal(loc.Local.HardwareAddr, dst)
		}
	}

	rx.parser = gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &rx.eth, &rx.dot1q, &rx.tlv)
	rx.parser.IgnoreUnsupported = true
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
			switch layerType {
			case layers.LayerTypeEthernet:
				if !rx.matchLL(rx.eth.SrcMAC, rx.eth.DstMAC) {
					continue DROP
				}
			case layers.LayerTypeDot1Q:
				// TODO match VLAN ID; recognize afpacket.AncillaryVLAN
			case ndnlayer.LayerTypeTLV:
				return copy(buf, rx.tlv.LayerContents()), nil
			}
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
