package ndn

import (
	"encoding/hex"
	"fmt"
	"math"

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

// Field encodes this KeyLocator.
func (kl KeyLocator) Field() tlv.Field {
	if len(kl.Name) > 0 && len(kl.Digest) > 0 {
		return tlv.FieldError(ErrKeyLocator)
	}
	if len(kl.Digest) > 0 {
		return tlv.TLV(an.TtKeyLocator, tlv.TLVBytes(an.TtKeyDigest, kl.Digest))
	}
	return tlv.TLVFrom(an.TtKeyLocator, kl.Name)
}

// UnmarshalBinary decodes from TLV-VALUE.
func (kl *KeyLocator) UnmarshalBinary(wire []byte) error {
	*kl = KeyLocator{}
	d := tlv.DecodingBuffer(wire)
	for _, de := range d.Elements() {
		switch de.Type {
		case an.TtName:
			if e := de.UnmarshalValue(&kl.Name); e != nil {
				return e
			}
		case an.TtKeyDigest:
			kl.Digest = de.Value
		default:
			if de.IsCriticalType() {
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

// EncodeAs creates a tlv.Fielder for either ISigInfo or DSigInfo TLV-TYPE.
// If si is nil, the encoding result contains SigType=SigNull.
func (si *SigInfo) EncodeAs(typ uint32) tlv.Fielder {
	return sigInfoFielder{typ, si}
}

// UnmarshalBinary decodes from TLV-VALUE.
func (si *SigInfo) UnmarshalBinary(wire []byte) (e error) {
	*si = SigInfo{}
	d := tlv.DecodingBuffer(wire)
	for _, de := range d.Elements() {
		switch de.Type {
		case an.TtSigType:
			if si.Type = uint32(unmarshalNNI(de, math.MaxUint32, &e, ErrSigType)); e != nil {
				return e
			}
		case an.TtKeyLocator:
			if e = de.UnmarshalValue(&si.KeyLocator); e != nil {
				return e
			}
		case an.TtSigNonce:
			if de.Length() < 1 {
				return ErrSigNonce
			}
			si.Nonce = de.Value
		case an.TtSigTime:
			if si.Time = unmarshalNNI(de, math.MaxUint64, &e, tlv.ErrRange); e != nil {
				return e
			}
		case an.TtSigSeqNum:
			if si.SeqNum = unmarshalNNI(de, math.MaxUint64, &e, tlv.ErrRange); e != nil {
				return e
			}
		default:
			if sigInfoExtensionTypes[de.Type] {
				si.Extensions = append(si.Extensions, de.Element)
			} else if de.IsCriticalType() {
				return tlv.ErrCritical
			}
		}
	}
	return d.ErrUnlessEOF()
}

func (si SigInfo) String() string {
	return fmt.Sprintf("%s:%v", an.SigTypeString(si.Type), si.KeyLocator)
}

type sigInfoFielder struct {
	typ uint32
	si  *SigInfo
}

func (sim sigInfoFielder) Field() tlv.Field {
	var fields []tlv.Fielder
	if si := sim.si; si == nil {
		fields = append(fields, tlv.TLVNNI(an.TtSigType, an.SigNull))
	} else {
		fields = append(fields, tlv.TLVNNI(an.TtSigType, uint64(si.Type)))
		if !si.KeyLocator.Empty() {
			fields = append(fields, si.KeyLocator)
		}
		if si.Time > 0 {
			fields = append(fields, tlv.TLVNNI(an.TtSigTime, si.Time))
		}
		if len(si.Nonce) > 0 {
			fields = append(fields, tlv.TLVBytes(an.TtSigNonce, si.Nonce))
		}
		if si.SeqNum > 0 {
			fields = append(fields, tlv.TLVNNI(an.TtSigSeqNum, si.SeqNum))
		}
		for _, extension := range si.Extensions {
			fields = append(fields, extension)
		}
	}
	return tlv.TLVFrom(sim.typ, fields...)
}

var sigInfoExtensionTypes = make(map[uint32]bool)

// RegisterSigInfoExtension registers an extension TLV-TYPE in SigInfo.
func RegisterSigInfoExtension(typ uint32) {
	sigInfoExtensionTypes[typ] = true
}
