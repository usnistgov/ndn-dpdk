package ndn

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding"
	"encoding/binary"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Interest represents an Interest packet.
type Interest struct {
	packet         *Packet
	Name           Name
	CanBePrefix    bool
	MustBeFresh    bool
	ForwardingHint ForwardingHint
	Nonce          Nonce
	Lifetime       time.Duration
	HopLimit       HopLimit
	AppParameters  []byte
	SigInfo        *SigInfo
	SigValue       []byte
}

var (
	_ tlv.Fielder                = Interest{}
	_ encoding.BinaryUnmarshaler = (*Interest)(nil)
)

// MakeInterest creates an Interest from flexible arguments.
// Arguments can contain:
//  - string or Name: set Name
//  - CanBePrefixFlag: set CanBePrefix
//  - MustBeFreshFlag: set MustBeFresh
//  - ForwardingHint: set forwarding hint
//  - Nonce: set Nonce
//  - time.Duration: set Lifetime
//  - HopLimit: set HopLimit
//  - []byte: set AppParameters
//  - LpL3: copy PitToken and CongMark
func MakeInterest(args ...any) (interest Interest) {
	packet := Packet{Interest: &interest}
	interest.packet = &packet
	for _, arg := range args {
		switch a := arg.(type) {
		case string:
			interest.Name = ParseName(a)
		case Name:
			interest.Name = a
		case tCanBePrefix:
			interest.CanBePrefix = true
		case tMustBeFresh:
			interest.MustBeFresh = true
		case ForwardingHint:
			interest.ForwardingHint = a
		case Nonce:
			interest.Nonce = a
		case time.Duration:
			interest.Lifetime = a
		case HopLimit:
			interest.HopLimit = a
		case []byte:
			interest.AppParameters = a
		case LpL3:
			packet.Lp.inheritFrom(a)
		default:
			panic("bad argument type " + reflect.TypeOf(arg).String())
		}
	}
	return interest
}

// ToPacket wraps Interest as Packet.
func (interest Interest) ToPacket() *Packet {
	if interest.packet == nil {
		interest.packet = &Packet{Interest: &interest}
	}
	return interest.packet
}

func (interest Interest) String() string {
	var b strings.Builder
	b.WriteString(interest.Name.String())
	if interest.CanBePrefix {
		b.WriteString("[P]")
	}
	if interest.MustBeFresh {
		b.WriteString("[F]")
	}
	return b.String()
}

// ApplyDefaultLifetime updates Lifetime to the default if it is not set.
func (interest *Interest) ApplyDefaultLifetime() time.Duration {
	if interest.Lifetime == 0 {
		interest.Lifetime = DefaultInterestLifetime
	}
	return interest.Lifetime
}

// UpdateParamsDigest appends or updates ParametersSha256DigestComponent.
// It will not remove erroneously present or duplicate ParametersSha256DigestComponent.
func (interest *Interest) UpdateParamsDigest() {
	paramsPortion, _ := tlv.EncodeFrom(interest.encodeParamsPortion()...)
	if len(paramsPortion) == 0 {
		return
	}
	digest := sha256.Sum256(paramsPortion)
	digestComp := MakeNameComponent(an.TtParametersSha256DigestComponent, digest[:])

	name, isReplaced := Name{}, false
	for _, comp := range interest.Name {
		if comp.Type == an.TtParametersSha256DigestComponent {
			name = append(name, digestComp)
			isReplaced = true
		} else {
			name = append(name, comp)
		}
	}

	if !isReplaced {
		name = append(name, digestComp)
	}
	interest.Name = name
}

// SignWith implements Signable interface.
// Caller should use signer.Sign(interest).
func (interest *Interest) SignWith(signer func(name Name, si *SigInfo) (LLSign, error)) error {
	if interest.SigInfo == nil {
		interest.SigInfo = newNullSigInfo()
	}
	llSign, e := signer(interest.Name, interest.SigInfo)
	if e != nil {
		return e
	}

	signedPortion, e := interest.encodeSignedPortion()
	if e != nil {
		return e
	}

	sig, e := llSign(signedPortion)
	if e != nil {
		return e
	}
	interest.SigValue = sig

	interest.UpdateParamsDigest()
	return nil
}

// VerifyWith implements Verifiable interface.
// Caller should use verifier.Verify(interest).
//
// This function cannot verify an Interest that contains unrecognized TLV elements.
func (interest Interest) VerifyWith(verifier func(name Name, si SigInfo) (LLVerify, error)) error {
	si := interest.SigInfo
	if si == nil {
		si = newNullSigInfo()
	}
	llVerify, e := verifier(interest.Name, *si)
	if e != nil {
		return e
	}

	signedPortion, e := interest.encodeSignedPortion()
	if e != nil {
		return e
	}
	return llVerify(signedPortion, interest.SigValue)
}

// Field implements tlv.Fielder interface.
func (interest Interest) Field() tlv.Field {
	fields := []tlv.Fielder{interest.Name}
	if interest.CanBePrefix {
		fields = append(fields, tlv.TLV(an.TtCanBePrefix))
	}
	if interest.MustBeFresh {
		fields = append(fields, tlv.TLV(an.TtMustBeFresh))
	}
	if len(interest.ForwardingHint) > 0 {
		fields = append(fields, interest.ForwardingHint)
	}

	nonce := interest.Nonce
	if nonce.IsZero() {
		nonce = NewNonce()
	}
	fields = append(fields, nonce)

	if lifetime := interest.Lifetime; lifetime != 0 && lifetime != DefaultInterestLifetime {
		if lifetime < MinInterestLifetime {
			return tlv.FieldError(ErrLifetime)
		}
		fields = append(fields, tlv.TLVNNI(an.TtInterestLifetime, lifetime/time.Millisecond))
	}
	if interest.HopLimit != 0 {
		fields = append(fields, interest.HopLimit)
	}
	fields = append(fields, interest.encodeParamsPortion()...)
	return tlv.TLVFrom(an.TtInterest, fields...)
}

// UnmarshalBinary decodes from TLV-VALUE.
func (interest *Interest) UnmarshalBinary(wire []byte) (e error) {
	*interest = Interest{}
	d := tlv.DecodingBuffer(wire)
	var paramsPortion []byte
	for _, de := range d.Elements() {
		switch de.Type {
		case an.TtName:
			if e = de.UnmarshalValue(&interest.Name); e != nil {
				return e
			}
		case an.TtCanBePrefix:
			interest.CanBePrefix = true
		case an.TtMustBeFresh:
			interest.MustBeFresh = true
		case an.TtForwardingHint:
			if e = de.UnmarshalValue(&interest.ForwardingHint); e != nil {
				return e
			}
		case an.TtNonce:
			if e = de.UnmarshalValue(&interest.Nonce); e != nil {
				return e
			}
		case an.TtInterestLifetime:
			if interest.Lifetime = time.Duration(de.UnmarshalNNI(uint64(math.MaxInt64/time.Millisecond), &e, ErrLifetime)); e != nil {
				return e
			}
			interest.Lifetime *= time.Millisecond
		case an.TtHopLimit:
			if e := de.UnmarshalValue(&interest.HopLimit); e != nil {
				return e
			}
		case an.TtAppParameters:
			interest.AppParameters = de.Value
			paramsPortion = de.WireAfter()
		case an.TtISigInfo:
			var si SigInfo
			if e := de.UnmarshalValue(&si); e != nil {
				return e
			}
			interest.SigInfo = &si
		case an.TtISigValue:
			interest.SigValue = de.Value
		default:
			if de.IsCriticalType() {
				return tlv.ErrCritical
			}
		}
	}

	if len(paramsPortion) > 0 {
		digest := sha256.Sum256(paramsPortion)
		foundParamsDigest := 0
		for i, comp := range interest.Name {
			if comp.Type != an.TtParametersSha256DigestComponent {
				continue
			}
			foundParamsDigest++
			if interest.SigInfo != nil && i != len(interest.Name)-1 {
				return ErrParamsDigest
			}
			if !hmac.Equal(digest[:], comp.Value) {
				return ErrParamsDigest
			}
		}
		if foundParamsDigest != 1 {
			return ErrParamsDigest
		}
	}

	return d.ErrUnlessEOF()
}

func (interest Interest) encodeParamsPortion() (fields []tlv.Fielder) {
	if len(interest.AppParameters) == 0 && interest.SigInfo == nil {
		return
	}
	fields = append(fields, tlv.TLVBytes(an.TtAppParameters, interest.AppParameters))
	if interest.SigInfo != nil {
		fields = append(fields, interest.SigInfo.EncodeAs(an.TtISigInfo), tlv.TLVBytes(an.TtISigValue, interest.SigValue))
	}
	return
}

func (interest Interest) encodeSignedPortion() (wire []byte, e error) {
	fields := make([]tlv.Fielder, 0, len(interest.Name)+2)

	nameWithoutDigest := interest.Name
	if interest.Name.Get(-1).Type == an.TtParametersSha256DigestComponent {
		nameWithoutDigest = interest.Name.GetPrefix(-1)
	}
	for _, comp := range nameWithoutDigest {
		fields = append(fields, comp)
	}

	fields = append(fields, tlv.TLVBytes(an.TtAppParameters, interest.AppParameters))
	fields = append(fields, interest.SigInfo.EncodeAs(an.TtISigInfo))
	return tlv.EncodeFrom(fields...)
}

// ForwardingHint represents a forwarding hint.
type ForwardingHint []Name

var (
	_ tlv.Fielder                = ForwardingHint{}
	_ encoding.BinaryUnmarshaler = (*ForwardingHint)(nil)
)

// Field implements tlv.Fielder interface.
func (fh ForwardingHint) Field() tlv.Field {
	subs := make([]tlv.Field, len(fh))
	for i, del := range fh {
		subs[i] = del.Field()
	}
	return tlv.TLV(an.TtForwardingHint, subs...)
}

// UnmarshalBinary decodes from TLV-VALUE.
func (fh *ForwardingHint) UnmarshalBinary(wire []byte) error {
	d := tlv.DecodingBuffer(wire)
	for _, de := range d.Elements() {
		switch de.Type {
		case an.TtName:
			var del Name
			if e := del.UnmarshalBinary(de.Value); e != nil {
				return e
			}
			*fh = append(*fh, del)
		default:
			if de.IsCriticalType() {
				return tlv.ErrCritical
			}
		}
	}
	return d.ErrUnlessEOF()
}

// Nonce represents an Interest Nonce.
type Nonce [4]byte

var (
	_ tlv.Fielder                = Nonce{}
	_ encoding.BinaryUnmarshaler = (*Nonce)(nil)
)

// NewNonce generates a random Nonce.
func NewNonce() (nonce Nonce) {
	rand.Read(nonce[:])
	return nonce
}

// NonceFromUint converts uint32 to Nonce, interpreted as big endian.
func NonceFromUint(n uint32) (nonce Nonce) {
	binary.BigEndian.PutUint32(nonce[:], n)
	return nonce
}

// IsZero returns true if the nonce is zero.
func (nonce Nonce) IsZero() bool {
	return nonce[0]|nonce[1]|nonce[2]|nonce[3] == 0
}

// ToUint converts Nonce to uint32, interpreted as big endian.
func (nonce Nonce) ToUint() uint32 {
	return binary.BigEndian.Uint32(nonce[:])
}

// Field implements tlv.Fielder interface.
func (nonce Nonce) Field() tlv.Field {
	return tlv.TLVBytes(an.TtNonce, nonce[:])
}

// UnmarshalBinary decodes from wire encoding.
func (nonce *Nonce) UnmarshalBinary(wire []byte) error {
	if len(wire) != len(*nonce) {
		return ErrNonceLen
	}
	copy(nonce[:], wire)
	return nil
}

// HopLimit represents a HopLimit field.
type HopLimit uint8

var (
	_ tlv.Fielder                = HopLimit(0)
	_ encoding.BinaryUnmarshaler = (*HopLimit)(nil)
)

// Field implements tlv.Fielder interface.
func (hl HopLimit) Field() tlv.Field {
	return tlv.TLVNNI(an.TtHopLimit, hl)
}

// UnmarshalBinary decodes from wire encoding.
func (hl *HopLimit) UnmarshalBinary(wire []byte) error {
	if len(wire) != 1 {
		return ErrHopLimit
	}
	*hl = HopLimit(wire[0])
	return nil
}

// Defaults and limits.
const (
	DefaultInterestLifetime time.Duration = 4000 * time.Millisecond
	MinInterestLifetime     time.Duration = 1 * time.Millisecond

	MinHopLimit = 1
	MaxHopLimit = math.MaxUint8
)

type tCanBePrefix bool
type tMustBeFresh bool

const (
	// CanBePrefixFlag enables CanBePrefix in MakeInterest.
	CanBePrefixFlag = tCanBePrefix(true)

	// MustBeFreshFlag enables MustBeFresh in MakeInterest.
	MustBeFreshFlag = tMustBeFresh(true)
)
