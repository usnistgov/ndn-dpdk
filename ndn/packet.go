// Package ndn implements Named Data Networking (NDN) packet semantics.
//
// This package contains the following important types:
//  Packet representation: Interest, Data, Nack, Packet
//  Low-level transport: Transport, L3Face
//  Security abstraction: Signer, Verifier
package ndn

import (
	"encoding/hex"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// L3Packet represents any NDN layer 3 packet.
type L3Packet interface {
	ToPacket() *Packet
}

// Packet represents an NDN layer 3 packet with associated LpL3.
type Packet struct {
	Lp       LpL3
	l3type   uint32
	l3value  []byte
	l3digest []byte
	Fragment *LpFragment
	Interest *Interest
	Data     *Data
	Nack     *Nack
}

func (pkt *Packet) String() string {
	suffix := ""
	if len(pkt.Lp.PitToken) != 0 {
		suffix = " token=" + hex.EncodeToString(pkt.Lp.PitToken)
	}
	switch {
	case pkt.Fragment != nil:
		return "Frag " + pkt.Fragment.String() + suffix
	case pkt.Interest != nil:
		return "I " + pkt.Interest.String() + suffix
	case pkt.Data != nil:
		return "D " + pkt.Data.String() + suffix
	case pkt.Nack != nil:
		return "N " + pkt.Nack.String() + suffix
	}
	return "(bad-NDN-packet)"
}

// ToPacket returns self.
func (pkt *Packet) ToPacket() *Packet {
	return pkt
}

// MarshalTlv encodes this packet.
func (pkt *Packet) MarshalTlv() (typ uint32, value []byte, e error) {
	if pkt.Fragment != nil {
		return pkt.Fragment.MarshalTlv()
	}

	header, payload, e := pkt.encodeL3()
	if e != nil {
		return 0, nil, e
	}

	if len(header) == 0 {
		return pkt.l3type, pkt.l3value, nil
	}
	return tlv.EncodeTlv(an.TtLpPacket, header, tlv.MakeElement(an.TtLpPayload, payload))
}

// UnmarshalTlv decodes from wire format.
func (pkt *Packet) UnmarshalTlv(typ uint32, value []byte) error {
	*pkt = Packet{}
	if typ != an.TtLpPacket {
		return pkt.decodeL3(typ, value)
	}

	d := tlv.Decoder(value)
	for _, field := range d.Elements() {
		switch field.Type {
		case an.TtPitToken:
			pkt.Lp.PitToken = field.Value
		case an.TtNack:
			pkt.Lp.NackReason = an.NackUnspecified
			d1 := tlv.Decoder(field.Value)
			for _, field1 := range d1.Elements() {
				switch field1.Type {
				case an.TtNackReason:
					if e := field1.UnmarshalNNI(&pkt.Lp.NackReason); e != nil {
						return e
					}
				default:
					if lpIsCritical(field1.Type) {
						return tlv.ErrCritical
					}
				}
			}
			if e := d1.ErrUnlessEOF(); e != nil {
				return e
			}
		case an.TtCongestionMark:
			if e := field.UnmarshalNNI(&pkt.Lp.CongMark); e != nil {
				return e
			}
		case an.TtLpPayload:
			d1 := tlv.Decoder(field.Value)
			field1, e := d1.Element()
			if e != nil {
				return e
			}
			e = pkt.decodeL3(field1.Type, field1.Value)
			if e != nil {
				return e
			}
			if e := d1.ErrUnlessEOF(); e != nil {
				return e
			}
		}
	}
	return d.ErrUnlessEOF()
}

func (pkt *Packet) encodeL3() (header, payload []byte, e error) {
	e = ErrFragment
	switch {
	case pkt.Interest != nil:
		pkt.l3type, pkt.l3value, e = pkt.Interest.MarshalTlv()
		pkt.l3digest = nil
		pkt.Lp.NackReason = an.NackNone
	case pkt.Data != nil:
		pkt.l3type, pkt.l3value, e = pkt.Data.MarshalTlv()
		pkt.l3digest = nil
		pkt.Lp.NackReason = an.NackNone
	case pkt.Nack != nil:
		pkt.l3type, pkt.l3value, e = pkt.Nack.Interest.MarshalTlv()
		pkt.l3digest = nil
		pkt.Lp.NackReason = pkt.Nack.Reason
	}
	if e != nil {
		return nil, nil, e
	}

	header, _ = tlv.Encode(pkt.Lp.encode())
	payload, _ = tlv.Encode(tlv.MakeElement(pkt.l3type, pkt.l3value))
	return header, payload, nil
}

func (pkt *Packet) decodeL3(typ uint32, value []byte) error {
	switch typ {
	case an.TtInterest:
		var interest Interest
		e := interest.UnmarshalBinary(value)
		if e != nil {
			return e
		}
		if pkt.Lp.NackReason != an.NackNone {
			var nack Nack
			nack.Reason = pkt.Lp.NackReason
			nack.Interest = interest
			nack.packet = pkt
			pkt.Nack = &nack
		} else {
			interest.packet = pkt
			pkt.Interest = &interest
		}
	case an.TtData:
		var data Data
		e := data.UnmarshalBinary(value)
		if e != nil {
			return e
		}
		data.packet = pkt
		pkt.Data = &data
	default:
		return ErrL3Type
	}

	pkt.l3type, pkt.l3value, pkt.l3digest = typ, value, nil
	return nil
}
