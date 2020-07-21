package ndn

import (
	"encoding/hex"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// KeyLocator represents KeyLocator in SignatureInfo.
type KeyLocator struct {
	Name   Name
	Digest []byte
}

// Empty returns true if KeyLocator has zero fields.
func (kl KeyLocator) Empty() bool {
	return len(kl.Name)+len(kl.Digest) == 0
}

// MarshalTlv encodes this KeyLocator.
func (kl KeyLocator) MarshalTlv() (typ uint32, value []byte, e error) {
	if len(kl.Name) > 0 && len(kl.Digest) > 0 {
		return 0, nil, ErrKeyLocator
	}
	if len(kl.Digest) > 0 {
		return tlv.EncodeTlv(an.TtKeyLocator, tlv.MakeElement(an.TtKeyDigest, kl.Digest))
	}
	return tlv.EncodeTlv(an.TtKeyLocator, kl.Name)
}

// UnmarshalBinary decodes from TLV-VALUE.
func (kl *KeyLocator) UnmarshalBinary(wire []byte) error {
	*kl = KeyLocator{}
	d := tlv.Decoder(wire)
	for _, field := range d.Elements() {
		switch field.Type {
		case an.TtName:
			if e := field.UnmarshalValue(&kl.Name); e != nil {
				return e
			}
		case an.TtKeyDigest:
			kl.Digest = field.Value
		default:
			if field.IsCriticalType() {
				return tlv.ErrCritical
			}
		}
	}

	if len(kl.Name) > 0 && len(kl.Digest) > 0 {
		return ErrKeyLocator
	}
	return d.ErrUnlessEOF()
}

func (kl KeyLocator) String() string {
	if len(kl.Digest) > 0 {
		return hex.EncodeToString(kl.Digest)
	}
	return kl.Name.String()
}

// SigInfo represents SignatureInfo on Interest or Data.
type SigInfo struct {
	Type       uint32
	KeyLocator KeyLocator
	Nonce      []byte
	Time       uint64
	SeqNum     uint64
	Extensions []tlv.Element
}

// EncodeAs creates an encodable object for either ISigInfo or DSigInfo TLV-TYPE.
// If si is nil, the encoding result contains SigType=SigNull.
func (si *SigInfo) EncodeAs(typ uint32) tlv.Marshaler {
	return sigInfoMarshaler{typ, si}
}

// UnmarshalBinary decodes from TLV-VALUE.
func (si *SigInfo) UnmarshalBinary(wire []byte) error {
	*si = SigInfo{}
	d := tlv.Decoder(wire)
	for _, field := range d.Elements() {
		switch field.Type {
		case an.TtSigType:
			if e := field.UnmarshalNNI(&si.Type); e != nil {
				return e
			}
		case an.TtKeyLocator:
			if e := field.UnmarshalValue(&si.KeyLocator); e != nil {
				return e
			}
		case an.TtSigNonce:
			if field.Length() < 1 {
				return ErrSigNonce
			}
			si.Nonce = field.Value
		case an.TtSigTime:
			if e := field.UnmarshalNNI(&si.Time); e != nil {
				return e
			}
		case an.TtSigSeqNum:
			if e := field.UnmarshalNNI(&si.SeqNum); e != nil {
				return e
			}
		default:
			if sigInfoExtensionTypes[field.Type] {
				si.Extensions = append(si.Extensions, field.Element)
			} else if field.IsCriticalType() {
				return tlv.ErrCritical
			}
		}
	}
	return d.ErrUnlessEOF()
}

func (si SigInfo) String() string {
	return fmt.Sprintf("%s:%v", an.SigTypeString(si.Type), si.KeyLocator)
}

type sigInfoMarshaler struct {
	typ uint32
	si  *SigInfo
}

func (sim sigInfoMarshaler) MarshalTlv() (typ uint32, value []byte, e error) {
	var fields []interface{}
	if si := sim.si; si == nil {
		fields = append(fields, tlv.MakeElementNNI(an.TtSigType, an.SigNull))
	} else {
		fields = append(fields, tlv.MakeElementNNI(an.TtSigType, si.Type))
		if !si.KeyLocator.Empty() {
			fields = append(fields, si.KeyLocator)
		}
		if si.Time > 0 {
			fields = append(fields, tlv.MakeElementNNI(an.TtSigTime, si.Time))
		}
		if len(si.Nonce) > 0 {
			fields = append(fields, tlv.MakeElement(an.TtSigNonce, si.Nonce))
		}
		if si.SeqNum > 0 {
			fields = append(fields, tlv.MakeElementNNI(an.TtSigSeqNum, si.SeqNum))
		}
		fields = append(fields, si.Extensions)
	}
	return tlv.EncodeTlv(sim.typ, fields...)
}

var sigInfoExtensionTypes = make(map[uint32]bool)

// RegisterSigInfoExtension registers an extension TLV-TYPE in SigInfo.
func RegisterSigInfoExtension(typ uint32) {
	sigInfoExtensionTypes[typ] = true
}
