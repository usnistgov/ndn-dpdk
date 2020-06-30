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
	packet      *Packet
	Name        Name
	ContentType ContentType
	Freshness   time.Duration
	Content     []byte
}

// MakeData creates a Data from flexible arguments.
// Arguments can contain:
// - string or Name: set Name
// - ContentType
// - time.Duration: set Freshness
// - []byte: set Content
// - LpL3: copy PitToken and CongMark
// - Interest or *Interest: copy Name, set FreshnessPeriod if Interest has MustBeFresh, inherit LpL3
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
		packet := Packet{Data: &data}
		data.packet = &packet
	}
	return data.packet
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

// CanSatisfy determins whether this Data can satisfy the given Interest.
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

// MarshalTlv encodes this Data.
func (data Data) MarshalTlv() (typ uint32, value []byte, e error) {
	var metaFields []interface{}
	if data.ContentType > 0 {
		metaFields = append(metaFields, data.ContentType)
	}
	if data.Freshness > 0 {
		metaFields = append(metaFields, tlv.MakeElementNNI(an.TtFreshnessPeriod, data.Freshness/time.Millisecond))
	}

	fields := []interface{}{data.Name}
	if len(metaFields) > 0 {
		metaV, e := tlv.Encode(metaFields...)
		if e != nil {
			return 0, nil, e
		}
		fields = append(fields, tlv.MakeElement(an.TtMetaInfo, metaV))
	}
	if len(data.Content) > 0 {
		fields = append(fields, tlv.MakeElement(an.TtContent, data.Content))
	}
	fields = append(fields, dataFakeSig)

	return tlv.EncodeTlv(an.TtData, fields)
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
		case an.TtDSigValue:
		default:
			if field.IsCriticalType() {
				return tlv.ErrCritical
			}
		}
	}
	return d.ErrUnlessEOF()
}

func (data Data) String() string {
	return data.Name.String()
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

var dataFakeSig []byte

func init() {
	sigType, _ := tlv.Encode(tlv.MakeElementNNI(an.TtSigType, an.SigHmacWithSha256))
	dataFakeSig, _ = tlv.Encode(
		tlv.MakeElement(an.TtDSigInfo, sigType),
		tlv.MakeElement(an.TtDSigValue, make([]byte, sha256.Size)),
	)
}
