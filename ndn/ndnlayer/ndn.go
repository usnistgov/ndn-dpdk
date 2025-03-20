package ndnlayer

import (
	"errors"

	"github.com/gopacket/gopacket"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// LayerTypeNDN identifies NDN layer.
var LayerTypeNDN = gopacket.RegisterLayerType(1636, gopacket.LayerTypeMetadata{
	Name:    "NDN",
	Decoder: gopacket.DecodeFunc(decodeNDN),
})

// NDN is the layer for NDN packets.
type NDN struct {
	Packet *ndn.Packet
	wire   []byte
}

var _ interface {
	gopacket.ApplicationLayer
	gopacket.DecodingLayer
	gopacket.SerializableLayer
} = &NDN{}

// LayerType returns LayerTypeNDN.
func (NDN) LayerType() gopacket.LayerType {
	return LayerTypeNDN
}

// LayerContents returns TLV bytes.
func (l *NDN) LayerContents() []byte {
	return l.wire
}

// LayerPayload returns TLV-VALUE of Interest ApplicationParameters or Data Content.
func (l *NDN) LayerPayload() []byte {
	switch {
	case l.Packet.Interest != nil:
		return l.Packet.Interest.AppParameters
	case l.Packet.Data != nil:
		return l.Packet.Data.Content
	}
	return nil
}

// Payload implements gopacket.ApplicationLayer interface.
func (l *NDN) Payload() []byte {
	return l.LayerPayload()
}

// DecodeFromBytes recognizes NDNLPv2 packet.
// Input must only contain one TLV element.
func (l *NDN) DecodeFromBytes(wire []byte, df gopacket.DecodeFeedback) error {
	l.Packet = &ndn.Packet{}
	if e := tlv.Decode(wire, l.Packet); e != nil {
		return e
	}

	l.wire = wire
	return nil
}

// CanDecode implements gopacket.DecodingLayer interface.
func (NDN) CanDecode() gopacket.LayerClass {
	return LayerTypeNDN
}

// NextLayerType implements gopacket.DecodingLayer interface.
func (NDN) NextLayerType() gopacket.LayerType {
	return gopacket.LayerTypePayload
}

// SerializeTo implements gopacket.SerializableLayer interface.
func (l *NDN) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {
	if l.Packet == nil {
		return errors.New("no Packet")
	}

	return encodeField(b, l.Packet)
}

func decodeNDN(wire []byte, p gopacket.PacketBuilder) error {
	l := &NDN{}
	if e := l.DecodeFromBytes(wire, p); e != nil {
		return e
	}
	p.AddLayer(l)
	p.SetApplicationLayer(l)
	return p.NextDecoder(gopacket.DecodePayload)
}
