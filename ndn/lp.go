package ndn

import (
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// LpHeader contains information in NDNLPv2 header.
type LpHeader struct {
	PitToken   []byte
	NackReason an.NackReason
	CongMark   int
}

// Empty returns true if LpHeader has zero fields.
func (lph LpHeader) Empty() bool {
	return len(lph.PitToken) == 0 && lph.NackReason == an.NackNone && lph.CongMark == 0
}

func (lph LpHeader) encode() (fields []interface{}) {
	if len(lph.PitToken) > 0 {
		fields = append(fields, tlv.MakeElement(an.TtPitToken, lph.PitToken))
	}
	if lph.NackReason != an.NackNone {
		nackV, _ := tlv.Encode(tlv.MakeElementNNI(an.TtNackReason, lph.NackReason))
		fields = append(fields, tlv.MakeElement(an.TtNack, nackV))
	}
	if lph.CongMark != 0 {
		fields = append(fields, tlv.MakeElementNNI(an.TtCongestionMark, lph.CongMark))
	}
	return fields
}

// Packet represents an NDN layer 3 packet with associated LpHeader.
type Packet struct {
	Lp       LpHeader
	l3type   uint32
	l3value  []byte
	l3digest []byte
	Interest *Interest
	Data     *Data
	Nack     *Nack
}

// MarshalTlv encodes this packet.
func (pkt *Packet) MarshalTlv() (typ uint32, value []byte, e error) {
	switch {
	case pkt.Interest != nil:
		pkt.l3type, pkt.l3value, e = pkt.Interest.MarshalTlv()
		pkt.l3digest = nil
		pkt.Lp.NackReason = an.NackNone
	case pkt.Data != nil:
		pkt.l3type, pkt.l3value, e = pkt.Interest.MarshalTlv()
		pkt.l3digest = nil
		pkt.Lp.NackReason = an.NackNone
	case pkt.Nack != nil:
		pkt.l3type, pkt.l3value, e = pkt.Nack.Interest.MarshalTlv()
		pkt.l3digest = nil
		pkt.Lp.NackReason = pkt.Nack.Reason
	}
	if e != nil {
		return 0, nil, e
	}
	if pkt.Lp.Empty() {
		return pkt.l3type, pkt.l3value, nil
	}
	lpPayload, e := tlv.Encode(tlv.MakeElement(pkt.l3type, pkt.l3value))
	if e != nil {
		return 0, nil, e
	}
	return tlv.EncodeTlv(an.TtLpPacket, pkt.Lp.encode(), tlv.MakeElement(an.TtLpPayload, lpPayload))
}

// UnmarshalTlv decodes from wire format.
func (pkt *Packet) UnmarshalTlv(typ uint32, value []byte) error {
	*pkt = Packet{}
	if an.TlvType(typ) != an.TtLpPacket {
		return pkt.decodeL3(typ, value)
	}

	d := tlv.Decoder(value)
	for _, field := range d.Elements() {
		switch an.TlvType(field.Type) {
		case an.TtPitToken:
			pkt.Lp.PitToken = field.Value
		case an.TtNack:
			d1 := tlv.Decoder(field.Value)
			for _, field1 := range d1.Elements() {
				switch an.TlvType(field1.Type) {
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

func (pkt *Packet) decodeL3(typ uint32, value []byte) error {
	switch an.TlvType(typ) {
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
			nack.Packet = pkt
			pkt.Nack = &nack
		} else {
			interest.Packet = pkt
			pkt.Interest = &interest
		}
	case an.TtData:
		var data Data
		e := data.UnmarshalBinary(value)
		if e != nil {
			return e
		}
		data.Packet = pkt
		pkt.Data = &data
	default:
		return ErrL3Type
	}

	pkt.l3type, pkt.l3value, pkt.l3digest = typ, value, nil
	return nil
}

func lpIsCritical(typ uint32) bool {
	return typ < 800 || typ > 959 && (typ&0x03) != 0
}
