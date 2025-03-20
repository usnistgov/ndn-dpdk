// Package ndn implements Named Data Networking (NDN) packet semantics.
// This is the top-level package of NDNgo, a minimal NDN library in pure Go.
//
// This package contains the following important types:
//
//	Packet representation:
//	- Interest
//	- Data
//	- Nack
//	- Packet
//
//	Security abstraction:
//	- Signer
//	- Verifier
package ndn

import (
	"encoding/binary"
	"encoding/hex"
	"math"

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

var _ interface {
	tlv.Fielder
	tlv.Unmarshaler
	L3Packet
} = &Packet{}

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

// Field implements tlv.Fielder interface.
func (pkt *Packet) Field() tlv.Field {
	if pkt.Fragment != nil {
		return pkt.Fragment.Field()
	}

	header, payload, e := pkt.encodeL3()
	if e != nil {
		return tlv.FieldError(e)
	}

	if len(header) == 0 {
		return tlv.Bytes(payload)
	}
	return tlv.TLV(an.TtLpPacket, tlv.Bytes(header), tlv.TLVBytes(an.TtLpPayload, payload))
}

func (pkt *Packet) encodeL3() (header, payload []byte, e error) {
	var l3fielder tlv.Fielder
	switch {
	case len(pkt.l3value) > 0:
		l3fielder = tlv.TLVBytes(pkt.l3type, pkt.l3value)
	case pkt.Interest != nil:
		l3fielder = pkt.Interest
		pkt.l3digest = nil
		pkt.Lp.NackReason = an.NackNone
	case pkt.Data != nil:
		l3fielder = pkt.Data
		pkt.l3digest = nil
		pkt.Lp.NackReason = an.NackNone
	case pkt.Nack != nil:
		l3fielder = pkt.Nack.Interest
		pkt.l3digest = nil
		pkt.Lp.NackReason = pkt.Nack.Reason
	default:
		return nil, nil, ErrFragment
	}

	if payload, e = tlv.EncodeFrom(l3fielder); e != nil {
		return nil, nil, e
	}
	if header, e = tlv.Encode(pkt.Lp.encode()...); e != nil {
		return nil, nil, e
	}
	return header, payload, nil
}

// UnmarshalTLV decodes from wire format.
func (pkt *Packet) UnmarshalTLV(typ uint32, value []byte) (e error) {
	*pkt = Packet{}
	if typ != an.TtLpPacket {
		return pkt.decodeL3(typ, value)
	}

	return pkt.decodeValue(value)
}

func (pkt *Packet) decodeValue(value []byte) (e error) {
	fragment := LpFragment{FragCount: 1}
	d := tlv.DecodingBuffer(value)
	for de := range d.IterElements() {
		switch de.Type {
		case an.TtLpSeqNum:
			if de.Length() != 8 {
				return ErrFragment
			}
			fragment.SeqNum = binary.BigEndian.Uint64(de.Value)
		case an.TtFragIndex:
			if fragment.FragIndex = int(de.UnmarshalNNI(math.MaxInt32, &e, ErrFragment)); e != nil {
				return e
			}
		case an.TtFragCount:
			if fragment.FragCount = int(de.UnmarshalNNI(math.MaxInt32, &e, ErrFragment)); e != nil {
				return e
			}
			if fragment.FragCount > 1 {
				pkt.Fragment = &fragment
			}
		case an.TtPitToken:
			pkt.Lp.PitToken = de.Value
		case an.TtNack:
			pkt.Lp.NackReason = an.NackUnspecified
			d1 := tlv.DecodingBuffer(de.Value)
			for _, de1 := range d1.Elements() {
				switch de1.Type {
				case an.TtNackReason:
					if pkt.Lp.NackReason = uint8(de1.UnmarshalNNI(math.MaxUint8, &e, tlv.ErrRange)); e != nil {
						return e
					}
				default:
					if lpIsCritical(de1.Type) {
						return tlv.ErrCritical
					}
				}
			}
			if e = d1.ErrUnlessEOF(); e != nil {
				return e
			}
		case an.TtCongestionMark:
			if pkt.Lp.CongMark = uint8(de.UnmarshalNNI(math.MaxUint8, &e, tlv.ErrRange)); e != nil {
				return e
			}
		case an.TtLpPayload:
			if e = pkt.decodePayload(de.Value); e != nil {
				return e
			}
		default:
			if lpIsCritical(de.Type) {
				return tlv.ErrCritical
			}
		}
	}
	if fragment.FragIndex >= fragment.FragCount {
		return ErrFragment
	}
	return d.ErrUnlessEOF()
}

func (pkt *Packet) decodePayload(payload []byte) error {
	if pkt.Fragment != nil {
		pkt.Fragment.Payload = payload
		return nil
	}

	d := tlv.DecodingBuffer(payload)
	field, e := d.Element()
	if e != nil {
		return e
	}
	if e := pkt.decodeL3(field.Type, field.Value); e != nil {
		return e
	}
	return d.ErrUnlessEOF()
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
