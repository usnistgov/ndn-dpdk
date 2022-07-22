// Package ndnlayer provides a GoPacket layer for NDN.
package ndnlayer

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Assigned numbers.
const (
	EthernetTypeNDN layers.EthernetType = an.EtherTypeNDN
	UDPPortNDN      layers.UDPPort      = an.UDPPortNDN
)

// LayerTypeTLV identifies NDN-TLV layer.
var LayerTypeTLV = gopacket.RegisterLayerType(1638, gopacket.LayerTypeMetadata{
	Name:    "NDN-TLV",
	Decoder: gopacket.DecodeFunc(decodeTLV),
})

// TLV is the layer for NDN-TLV.
type TLV struct {
	Element tlv.Element
	wire    []byte
}

var (
	_ gopacket.Layer         = (*TLV)(nil)
	_ gopacket.DecodingLayer = (*TLV)(nil)
)

// LayerType returns LayerTypeTLV.
func (TLV) LayerType() gopacket.LayerType {
	return LayerTypeTLV
}

// LayerContents returns TLV bytes.
func (l *TLV) LayerContents() []byte {
	return l.wire
}

// LayerPayload returns TLV bytes.
func (l *TLV) LayerPayload() []byte {
	return l.wire
}

// DecodeFromBytes recognizes NDN-TLV structure.
// Input must start with an TLV element, and may contain padding at the end.
func (l *TLV) DecodeFromBytes(wire []byte, df gopacket.DecodeFeedback) error {
	d := tlv.DecodingBuffer(wire)
	de, e := d.Element()
	if e != nil {
		return e
	}

	l.Element = de.Element
	l.wire = de.Wire
	return nil
}

// CanDecode implements gopacket.DecodingLayer interface.
func (TLV) CanDecode() gopacket.LayerClass {
	return LayerTypeTLV
}

// NextLayerType implements gopacket.DecodingLayer interface.
func (TLV) NextLayerType() gopacket.LayerType {
	return LayerTypeNDN
}

func decodeTLV(wire []byte, p gopacket.PacketBuilder) error {
	l := &TLV{}
	if e := l.DecodeFromBytes(wire, p); e != nil {
		return e
	}
	p.AddLayer(l)
	return p.NextDecoder(LayerTypeNDN)
}

func init() {
	layers.EthernetTypeMetadata[EthernetTypeNDN] = layers.EnumMetadata{
		DecodeWith: gopacket.DecodeFunc(decodeTLV),
		Name:       LayerTypeTLV.String(),
		LayerType:  LayerTypeTLV,
	}

	layers.RegisterUDPPortLayerType(UDPPortNDN, LayerTypeTLV)
}

func encodeField(b gopacket.SerializeBuffer, fields ...tlv.Fielder) error {
	wire, e := tlv.EncodeFrom(fields...)
	if e != nil {
		return e
	}

	room, e := b.PrependBytes(len(wire))
	copy(room, wire)
	return e
}

// SerializeFrom creates a gopacket.SerializableLayer from a sequence of tlv.Fielders.
func SerializeFrom(fields ...tlv.Fielder) gopacket.SerializableLayer {
	return serializableFielders(fields)
}

type serializableFielders []tlv.Fielder

func (fields serializableFielders) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
	return encodeField(b, fields...)
}

func (serializableFielders) LayerType() gopacket.LayerType {
	return LayerTypeTLV
}
