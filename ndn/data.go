package ndn

import (
	"crypto/sha256"
	"reflect"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Data represents a Data packet.
type Data struct {
	packet           *Packet
	l3SigValueOffset int
	Name             Name
	ContentType      ContentType
	Freshness        time.Duration
	Content          []byte
	SigInfo          *SigInfo
	SigValue         []byte
}

// MakeData creates a Data from flexible arguments.
// Arguments can contain:
//  - string or Name: set Name
//  - ContentType
//  - time.Duration: set Freshness
//  - []byte: set Content
//  - LpL3: copy PitToken and CongMark
//  - Interest or *Interest: copy Name, set FreshnessPeriod if Interest has MustBeFresh, inherit LpL3
func MakeData(args ...interface{}) (data Data) {
	packet := Packet{Data: &data}
	data.packet = &packet
	handleInterestArg := func(a *Interest) {
		data.Name = a.Name
		if a.MustBeFresh {
			data.Freshness = 1 * time.Millisecond
		}
		if ipkt := a.packet; ipkt != nil {
			packet.Lp.inheritFrom(ipkt.Lp)
		}
	}
	for _, arg := range args {
		switch a := arg.(type) {
		case string:
			data.Name = ParseName(a)
		case Name:
			data.Name = a
		case ContentType:
			data.ContentType = a
		case time.Duration:
			data.Freshness = a
		case []byte:
			data.Content = a
		case LpL3:
			packet.Lp.inheritFrom(a)
		case Interest:
			handleInterestArg(&a)
		case *Interest:
			handleInterestArg(a)
		default:
			panic("bad argument type " + reflect.TypeOf(arg).String())
		}
	}
	return data
}

// ToPacket wraps Data as Packet.
func (data Data) ToPacket() *Packet {
	if data.packet == nil {
		data.packet = &Packet{Data: &data}
	}
	return data.packet
}

func (data Data) String() string {
	return data.Name.String()
}

// ComputeDigest computes implicit digest of this Data.
//
// If data was decoded from Packet (data.packet is assigned), the digest is of the origin packet.
// Computed digest is cached on data.packet.
// Modifying a decoded Data will cause this function to return incorrect digest.
//
// If data was constructed (data.packet is unassigned), the digest is of the encoding of the current packet,
// and is not cached.
func (data Data) ComputeDigest() []byte {
	if data.packet == nil {
		data.packet = new(Packet)
		data.packet.Data = &data
	}
	if data.packet.l3type != an.TtData {
		data.packet.l3type, data.packet.l3value, _ = data.MarshalTlv()
	}
	if data.packet.l3digest == nil {
		wire, _ := tlv.Encode(tlv.MakeElement(data.packet.l3type, data.packet.l3value))
		digest := sha256.Sum256(wire)
		data.packet.l3digest = digest[:]
	}
	return data.packet.l3digest
}

// FullName returns full name of this Data.
func (data Data) FullName() Name {
	fullName := make(Name, len(data.Name)+1)
	i := copy(fullName, data.Name)
	fullName[i] = MakeNameComponent(an.TtImplicitSha256DigestComponent, data.ComputeDigest())
	return fullName
}

// CanSatisfy determines whether this Data can satisfy the given Interest.
func (data Data) CanSatisfy(interest Interest) bool {
	switch {
	case len(interest.Name) == 0: // invalid Interest
		return false
	case interest.MustBeFresh && data.Freshness <= 0:
		return false
	case interest.Name[len(interest.Name)-1].Type == an.TtImplicitSha256DigestComponent:
		return interest.Name.Equal(data.FullName())
	case interest.CanBePrefix:
		return interest.Name.IsPrefixOf(data.Name)
	default:
		return interest.Name.Equal(data.Name)
	}
}

// SignWith implements Signable interface.
// Caller should use signer.Sign(data).
func (data *Data) SignWith(signer func(name Name, si *SigInfo) (LLSign, error)) error {
	if data.SigInfo == nil {
		data.SigInfo = newNullSigInfo()
	}
	llSign, e := signer(data.Name, data.SigInfo)
	if e != nil {
		return e
	}

	signedPortion, e := data.encodeSignedPortion()
	if e != nil {
		return e
	}

	sig, e := llSign(signedPortion)
	if e != nil {
		return e
	}
	data.SigValue = sig
	return nil
}

// VerifyWith implements Verifiable interface.
// Caller should use verifier.Verify(data).
//
// If data was decoded from Packet (data.packet is assigned), verification is on the origin packet.
// Modifying a decoded Data will cause this function to return incorrect result.
//
// If data was constructed (data.packet is unassigned), verification is on the encoding of the current packet.
func (data Data) VerifyWith(verifier func(name Name, si SigInfo) (LLVerify, error)) error {
	si := data.SigInfo
	if si == nil {
		si = newNullSigInfo()
	}
	llVerify, e := verifier(data.Name, *si)
	if e != nil {
		return e
	}

	var signedPortion []byte
	if data.packet != nil && data.l3SigValueOffset > 0 {
		signedPortion = data.packet.l3value[:data.l3SigValueOffset]
	} else {
		signedPortion, e = data.encodeSignedPortion()
		if e != nil {
			return e
		}
	}
	return llVerify(signedPortion, data.SigValue)
}

// MarshalTlv encodes this Data.
func (data Data) MarshalTlv() (typ uint32, value []byte, e error) {
	signedPortion, e := data.encodeSignedPortion()
	if e != nil {
		return 0, nil, e
	}
	return tlv.EncodeTlv(an.TtData, signedPortion, tlv.MakeElement(an.TtDSigValue, data.SigValue))
}

// UnmarshalBinary decodes from TLV-VALUE.
func (data *Data) UnmarshalBinary(wire []byte) error {
	*data = Data{}
	d := tlv.Decoder(wire)
	for _, field := range d.Elements() {
		switch field.Type {
		case an.TtName:
			if e := field.UnmarshalValue(&data.Name); e != nil {
				return e
			}
		case an.TtMetaInfo:
			d1 := tlv.Decoder(field.Value)
			for _, field1 := range d1.Elements() {
				switch field1.Type {
				case an.TtContentType:
					if e := field1.UnmarshalValue(&data.ContentType); e != nil {
						return e
					}
				case an.TtFreshnessPeriod:
					if e := field1.UnmarshalNNI(&data.Freshness); e != nil {
						return e
					}
					data.Freshness *= time.Millisecond
				}
			}
			if e := d1.ErrUnlessEOF(); e != nil {
				return e
			}
		case an.TtContent:
			data.Content = field.Value
		case an.TtDSigInfo:
			var si SigInfo
			if e := field.UnmarshalValue(&si); e != nil {
				return e
			}
			data.SigInfo = &si
		case an.TtDSigValue:
			data.SigValue = field.Value
			data.l3SigValueOffset = len(wire) - len(field.WireAfter())
		default:
			if field.IsCriticalType() {
				return tlv.ErrCritical
			}
		}
	}
	return d.ErrUnlessEOF()
}

func (data Data) encodeSignedPortion() (wire []byte, e error) {
	fields := []interface{}{data.Name}

	var metaFields []interface{}
	if data.ContentType > 0 {
		metaFields = append(metaFields, data.ContentType)
	}
	if data.Freshness > 0 {
		metaFields = append(metaFields, tlv.MakeElementNNI(an.TtFreshnessPeriod, data.Freshness/time.Millisecond))
	}
	if len(metaFields) > 0 {
		metaV, e := tlv.Encode(metaFields...)
		if e != nil {
			return nil, e
		}
		fields = append(fields, tlv.MakeElement(an.TtMetaInfo, metaV))
	}

	if len(data.Content) > 0 {
		fields = append(fields, tlv.MakeElement(an.TtContent, data.Content))
	}
	fields = append(fields, data.SigInfo.EncodeAs(an.TtDSigInfo))
	return tlv.Encode(fields...)
}

// ContentType represents a ContentType field.
type ContentType uint

// MarshalTlv encodes this ContentType.
func (ct ContentType) MarshalTlv() (typ uint32, value []byte, e error) {
	return tlv.EncodeTlv(an.TtContentType, tlv.NNI(ct))
}

// UnmarshalBinary decodes from wire encoding.
func (ct *ContentType) UnmarshalBinary(wire []byte) error {
	var n tlv.NNI
	e := n.UnmarshalBinary(wire)
	*ct = ContentType(n)
	return e
}
