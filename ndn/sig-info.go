package ndn

import (
	"encoding"
	"encoding/hex"
	"fmt"
	"math"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"github.com/zyedidia/generic/mapset"
)

// KeyLocator represents KeyLocator in SignatureInfo.
type KeyLocator struct {
	Name   Name
	Digest []byte
}

var (
	_ tlv.Fielder                = KeyLocator{}
	_ encoding.BinaryUnmarshaler = &KeyLocator{}
)

// Empty returns true if KeyLocator has zero fields.
func (kl KeyLocator) Empty() bool {
	return len(kl.Name)+len(kl.Digest) == 0
}

// Field implements tlv.Fielder interface.
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
	for de := range d.IterElements() {
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

var _ encoding.BinaryUnmarshaler = &SigInfo{}

// EncodeAs creates a tlv.Fielder for either ISigInfo or DSigInfo TLV-TYPE.
// If si is nil, the encoding result contains SigType=SigNull.
func (si *SigInfo) EncodeAs(typ uint32) tlv.Fielder {
	return sigInfoFielder{typ, si}
}

// FindExtension retrieves extension field by TLV-TYPE number.
func (si *SigInfo) FindExtension(typ uint32) *tlv.Element {
	for _, ext := range si.Extensions {
		if ext.Type == typ {
			return &ext
		}
	}
	return nil
}

// UnmarshalBinary decodes from TLV-VALUE.
func (si *SigInfo) UnmarshalBinary(wire []byte) (e error) {
	*si = SigInfo{}
	d := tlv.DecodingBuffer(wire)
	for de := range d.IterElements() {
		switch de.Type {
		case an.TtSigType:
			if si.Type = uint32(de.UnmarshalNNI(math.MaxUint32, &e, ErrSigType)); e != nil {
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
			if si.Time = de.UnmarshalNNI(math.MaxUint64, &e, tlv.ErrRange); e != nil {
				return e
			}
		case an.TtSigSeqNum:
			if si.SeqNum = de.UnmarshalNNI(math.MaxUint64, &e, tlv.ErrRange); e != nil {
				return e
			}
		default:
			if sigInfoExtensionTypes.Has(de.Type) {
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
		fields = append(fields, tlv.TLVNNI(an.TtSigType, si.Type))
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

var sigInfoExtensionTypes = mapset.New[uint32]()

// RegisterSigInfoExtension registers an extension TLV-TYPE in SigInfo.
func RegisterSigInfoExtension(typ uint32) {
	sigInfoExtensionTypes.Put(typ)
}
