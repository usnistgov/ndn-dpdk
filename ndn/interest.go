package ndn

import (
	"crypto/hmac"
	"crypto/sha256"
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

// MakeInterest creates an Interest from flexible arguments.
// Arguments can contain:
//  - string or Name: set Name
//  - CanBePrefixFlag: set CanBePrefix
//  - MustBeFreshFlag: set MustBeFresh
//  - FHDelegation: append forwarding hint delegation
//  - Nonce: set Nonce
//  - time.Duration: set Lifetime
//  - HopLimit: set HopLimit
//  - []byte: set AppParameters
//  - LpL3: copy PitToken and CongMark
func MakeInterest(args ...interface{}) (interest Interest) {
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
		case FHDelegation:
			interest.ForwardingHint = append(interest.ForwardingHint, a)
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
	paramsPortion, _ := tlv.Encode(interest.encodeParamsPortion())
	if len(paramsPortion) == 0 {
		return
	}
	digest := sha256.Sum256(paramsPortion)

	for _, comp := range interest.Name {
		if comp.Type == an.TtParametersSha256DigestComponent {
			comp.Value = digest[:]
			return
		}
	}

	interest.Name = append(interest.Name, MakeNameComponent(an.TtParametersSha256DigestComponent, digest[:]))
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

// MarshalTlv encodes this Interest.
func (interest Interest) MarshalTlv() (typ uint32, value []byte, e error) {
	fields := []interface{}{interest.Name}
	if interest.CanBePrefix {
		fields = append(fields, tlv.MakeElement(an.TtCanBePrefix, nil))
	}
	if interest.MustBeFresh {
		fields = append(fields, tlv.MakeElement(an.TtMustBeFresh, nil))
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
			return 0, nil, ErrLifetime
		}
		fields = append(fields, tlv.MakeElementNNI(an.TtInterestLifetime, lifetime/time.Millisecond))
	}
	if interest.HopLimit != 0 {
		fields = append(fields, interest.HopLimit)
	}
	fields = append(fields, interest.encodeParamsPortion())
	return tlv.EncodeTlv(an.TtInterest, fields)
}

// UnmarshalBinary decodes from TLV-VALUE.
func (interest *Interest) UnmarshalBinary(wire []byte) error {
	*interest = Interest{}
	d := tlv.Decoder(wire)
	var paramsPortion []byte
	for _, field := range d.Elements() {
		switch field.Type {
		case an.TtName:
			if e := field.UnmarshalValue(&interest.Name); e != nil {
				return e
			}
		case an.TtCanBePrefix:
			interest.CanBePrefix = true
		case an.TtMustBeFresh:
			interest.MustBeFresh = true
		case an.TtForwardingHint:
			if e := field.UnmarshalValue(&interest.ForwardingHint); e != nil {
				return e
			}
		case an.TtNonce:
			if e := field.UnmarshalValue(&interest.Nonce); e != nil {
				return e
			}
		case an.TtInterestLifetime:
			if e := field.UnmarshalNNI(&interest.Lifetime); e != nil {
				return e
			}
			interest.Lifetime *= time.Millisecond
		case an.TtHopLimit:
			if e := field.UnmarshalValue(&interest.HopLimit); e != nil {
				return e
			}
		case an.TtAppParameters:
			interest.AppParameters = field.Value
			paramsPortion = field.WireAfter()
		case an.TtISigInfo:
			var si SigInfo
			if e := field.UnmarshalValue(&si); e != nil {
				return e
			}
			interest.SigInfo = &si
		case an.TtISigValue:
			interest.SigValue = field.Value
		default:
			if field.IsCriticalType() {
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

func (interest Interest) encodeParamsPortion() (fields []interface{}) {
	if len(interest.AppParameters) == 0 && interest.SigInfo == nil {
		return
	}
	fields = []interface{}{tlv.MakeElement(an.TtAppParameters, interest.AppParameters)}
	if interest.SigInfo != nil {
		fields = append(fields, interest.SigInfo.EncodeAs(an.TtISigInfo), tlv.MakeElement(an.TtISigValue, interest.SigValue))
	}
	return
}

func (interest Interest) encodeSignedPortion() (wire []byte, e error) {
	var fields []interface{}
	if interest.Name.Get(-1).Type == an.TtParametersSha256DigestComponent {
		fields = append(fields, []NameComponent(interest.Name.GetPrefix(-1)))
	} else {
		fields = append(fields, []NameComponent(interest.Name))
	}
	fields = append(fields, tlv.MakeElement(an.TtAppParameters, interest.AppParameters), interest.SigInfo.EncodeAs(an.TtISigInfo))
	return tlv.Encode(fields...)
}

// ForwardingHint represents a forwarding hint.
type ForwardingHint []FHDelegation

// Append adds a delegation.
// name should be either Name or string.
func (fh *ForwardingHint) Append(preference int, name interface{}) {
	*fh = append(*fh, MakeFHDelegation(preference, name))
}

// MarshalTlv encodes this forwarding hint.
func (fh ForwardingHint) MarshalTlv() (typ uint32, value []byte, e error) {
	return tlv.EncodeTlv(an.TtForwardingHint, []FHDelegation(fh))
}

// UnmarshalBinary decodes from TLV-VALUE.
func (fh *ForwardingHint) UnmarshalBinary(wire []byte) error {
	d := tlv.Decoder(wire)
	for _, field := range d.Elements() {
		switch field.Type {
		case an.TtDelegation:
			var del FHDelegation
			if e := del.UnmarshalBinary(field.Value); e != nil {
				return e
			}
			*fh = append(*fh, del)
		default:
			if field.IsCriticalType() {
				return tlv.ErrCritical
			}
		}
	}
	return d.ErrUnlessEOF()
}

// FHDelegation represents a delegation of forwarding hint.
type FHDelegation struct {
	Preference int
	Name       Name
}

// MakeFHDelegation creates a delegation.
// name should be either Name or string.
func MakeFHDelegation(preference int, name interface{}) (del FHDelegation) {
	del.Preference = preference
	switch a := name.(type) {
	case string:
		del.Name = ParseName(a)
	case Name:
		del.Name = a
	default:
		panic(reflect.TypeOf(name))
	}
	return del
}

// MarshalTlv encodes this delegation.
func (del FHDelegation) MarshalTlv() (typ uint32, value []byte, e error) {
	value, e = tlv.Encode(
		tlv.MakeElementNNI(an.TtPreference, del.Preference),
		del.Name,
	)
	return uint32(an.TtDelegation), value, e
}

// UnmarshalBinary decodes from TLV-VALUE.
func (del *FHDelegation) UnmarshalBinary(wire []byte) error {
	d := tlv.Decoder(wire)
	for _, field := range d.Elements() {
		switch field.Type {
		case an.TtPreference:
			if e := field.UnmarshalNNI(&del.Preference); e != nil {
				return e
			}
		case an.TtName:
			if e := field.UnmarshalValue(&del.Name); e != nil {
				return e
			}
		default:
			if field.IsCriticalType() {
				return tlv.ErrCritical
			}
		}
	}
	return d.ErrUnlessEOF()
}

// Nonce represents an Interest Nonce.
type Nonce [4]byte

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
	return (nonce[0] | nonce[1] | nonce[2] | nonce[3]) == 0
}

// ToUint converts Nonce to uint32, interpreted as big endian.
func (nonce Nonce) ToUint() uint32 {
	return binary.BigEndian.Uint32(nonce[:])
}

// MarshalTlv encodes this Nonce.
func (nonce Nonce) MarshalTlv() (typ uint32, value []byte, e error) {
	return uint32(an.TtNonce), nonce[:], nil
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

// MarshalTlv encodes this HopLimit.
func (hl HopLimit) MarshalTlv() (typ uint32, value []byte, e error) {
	return tlv.EncodeTlv(an.TtHopLimit, tlv.NNI(hl))
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
	// CanBePrefixFlag enables CanBePrefix in NewInterest.
	CanBePrefixFlag = tCanBePrefix(true)

	// MustBeFreshFlag enables MustBeFresh in NewInterest.
	MustBeFreshFlag = tMustBeFresh(true)
)
