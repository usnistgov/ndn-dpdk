package ndn

import (
	"crypto/sha256"
	"encoding"
	"math"
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
	FinalBlock       NameComponent
	Content          []byte
	SigInfo          *SigInfo
	SigValue         []byte
}

var (
	_ interface {
		tlv.Fielder
		L3Packet
	} = Data{}
	_ encoding.BinaryUnmarshaler = &Data{}
)

// MakeData creates a Data from flexible arguments.
// Arguments can contain:
//   - string or Name: set Name
//   - ContentType
//   - time.Duration: set Freshness
//   - FinalBlock: set FinalBlock
//   - FinalBlockFlag: set FinalBlock to the last name component, ignored if name is empty
//   - []byte: set Content
//   - LpL3: copy PitToken and CongMark
//   - Interest or *Interest: copy Name, set FreshnessPeriod if Interest has MustBeFresh, inherit LpL3
func MakeData(args ...any) (data Data) {
	data.packet = &Packet{}
	hasFinalBlockFlag := false
	handleInterestArg := func(a *Interest) {
		data.Name = a.Name
		if a.MustBeFresh {
			data.Freshness = 1 * time.Millisecond
		}
		if ipkt := a.packet; ipkt != nil {
			data.packet.Lp.inheritFrom(ipkt.Lp)
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
		case FinalBlock:
			data.FinalBlock = NameComponent(a)
		case tFinalBlockFlag:
			hasFinalBlockFlag = true
		case []byte:
			data.Content = a
		case LpL3:
			data.packet.Lp.inheritFrom(a)
		case Interest:
			handleInterestArg(&a)
		case *Interest:
			handleInterestArg(a)
		default:
			panic("bad argument type " + reflect.TypeOf(arg).String())
		}
	}
	if hasFinalBlockFlag && len(data.Name) > 0 {
		data.FinalBlock = data.Name.Get(-1)
	}
	return data
}

// ToPacket wraps Data as Packet.
func (data Data) ToPacket() (packet *Packet) {
	packet = &Packet{}
	if data.packet != nil {
		*packet = *data.packet
	}
	packet.Data = &data
	return packet
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
	var e error

	if data.packet == nil {
		data.packet = &Packet{Data: &data}
	}

	if data.packet.l3type != an.TtData {
		if data.packet.l3value, e = tlv.EncodeValueOnly(data); e != nil {
			return nil
		}
		data.packet.l3type = an.TtData
	}

	if data.packet.l3digest == nil {
		wire, e := tlv.Encode(tlv.TLVBytes(an.TtData, data.packet.l3value))
		if e != nil {
			return nil
		}
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

// IsFinalBlock determines whether FinalBlock field equals the last name component.
func (data Data) IsFinalBlock() bool {
	return data.FinalBlock.Valid() && data.FinalBlock.Equal(data.Name.Get(-1))
}

// CanSatisfy determines whether this Data can satisfy the given Interest.
func (data Data) CanSatisfy(interest Interest, optionalFlag ...CanSatisfyFlag) bool {
	var flag CanSatisfyFlag
	if len(optionalFlag) == 1 {
		flag = optionalFlag[0]
	}

	switch {
	case len(interest.Name) == 0: // invalid Interest
		return false
	case flag&CanSatisfyInCache != 0 && interest.MustBeFresh && data.Freshness <= 0:
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

// Field implements tlv.Fielder interface.
func (data Data) Field() tlv.Field {
	signedPortion, e := data.encodeSignedPortion()
	if e != nil {
		return tlv.FieldError(e)
	}
	return tlv.TLV(an.TtData, tlv.Bytes(signedPortion), tlv.TLVBytes(an.TtDSigValue, data.SigValue))
}

// UnmarshalBinary decodes from TLV-VALUE.
func (data *Data) UnmarshalBinary(value []byte) (e error) {
	*data = Data{}
	d := tlv.DecodingBuffer(value)
	for de := range d.IterElements() {
		switch de.Type {
		case an.TtName:
			if e = de.UnmarshalValue(&data.Name); e != nil {
				return e
			}
		case an.TtMetaInfo:
			if e = data.decodeMetaInfo(de.Value); e != nil {
				return e
			}
		case an.TtContent:
			data.Content = de.Value
		case an.TtDSigInfo:
			var si SigInfo
			if e = de.UnmarshalValue(&si); e != nil {
				return e
			}
			data.SigInfo = &si
		case an.TtDSigValue:
			data.SigValue = de.Value
			data.l3SigValueOffset = len(value) - len(de.WireAfter())
		default:
			if de.IsCriticalType() {
				return tlv.ErrCritical
			}
		}
	}
	return d.ErrUnlessEOF()
}

func (data *Data) decodeMetaInfo(value []byte) (e error) {
	d := tlv.DecodingBuffer(value)
	for de := range d.IterElements() {
		switch de.Type {
		case an.TtContentType:
			if data.ContentType = ContentType(de.UnmarshalNNI(math.MaxUint64, &e, tlv.ErrRange)); e != nil {
				return e
			}
		case an.TtFreshnessPeriod:
			if data.Freshness = time.Duration(de.UnmarshalNNI(uint64(math.MaxInt64/time.Millisecond), &e, tlv.ErrRange)); e != nil {
				return e
			}
			data.Freshness *= time.Millisecond
		case an.TtFinalBlock:
			if data.FinalBlock, e = DecodeFinalBlock(de); e != nil {
				return e
			}
		default:
			if de.IsCriticalType() {
				return tlv.ErrCritical
			}
		}
	}
	return d.ErrUnlessEOF()
}

func (data Data) encodeSignedPortion() (wire []byte, e error) {
	fields := []tlv.Fielder{data.Name}

	var metaFields []tlv.Field
	if data.ContentType > 0 {
		metaFields = append(metaFields, data.ContentType.Field())
	}
	if data.Freshness > 0 {
		metaFields = append(metaFields, tlv.TLVNNI(an.TtFreshnessPeriod, data.Freshness/time.Millisecond))
	}
	if data.FinalBlock.Valid() {
		metaFields = append(metaFields, tlv.TLVFrom(an.TtFinalBlock, data.FinalBlock))
	}
	if len(metaFields) > 0 {
		fields = append(fields, tlv.TLV(an.TtMetaInfo, metaFields...))
	}

	if len(data.Content) > 0 {
		fields = append(fields, tlv.TLVBytes(an.TtContent, data.Content))
	}
	fields = append(fields, data.SigInfo.EncodeAs(an.TtDSigInfo))
	return tlv.EncodeFrom(fields...)
}

// ContentType represents a ContentType field.
type ContentType uint64

func (ct ContentType) Field() tlv.Field {
	return tlv.TLVNNI(an.TtContentType, ct)
}

// UnmarshalBinary decodes from wire encoding.
func (ct *ContentType) UnmarshalBinary(wire []byte) error {
	var n tlv.NNI
	e := n.UnmarshalBinary(wire)
	*ct = ContentType(n)
	return e
}

// FinalBlock is passed to MakeData to set FinalBlock field.
type FinalBlock NameComponent

type tFinalBlockFlag bool

// FinalBlockFlag enables MakeData to set FinalBlock to the last name component.
const FinalBlockFlag = tFinalBlockFlag(true)

// DecodeFinalBlock decodes FinalBlock name component from FinalBlock TLV element.
func DecodeFinalBlock(de tlv.DecodingElement) (finalBlock NameComponent, e error) {
	if e = tlv.Decode(de.Value, &finalBlock); e != nil {
		return NameComponent{}, e
	}
	if !finalBlock.Valid() {
		return NameComponent{}, ErrComponentType
	}
	return finalBlock, nil
}

// Data.CanSatisfy flags.
type CanSatisfyFlag int

const (
	CanSatisfyInCache = 1 << iota
)
