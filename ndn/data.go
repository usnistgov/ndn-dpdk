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
	Packet      *Packet
	Name        Name
	ContentType ContentType
	Freshness   time.Duration
	Content     []byte
}

// MakeData creates a Data from flexible arguments.
// Arguments can contain string (as Name), Name, ContentType, time.Duration (as Lifetime),
// and []byte (as Content).
func MakeData(args ...interface{}) (data Data) {
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
		default:
			panic("bad argument type " + reflect.TypeOf(arg).String())
		}
	}
	return data
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
		switch an.TlvType(field.Type) {
		case an.TtName:
			if e := field.UnmarshalValue(&data.Name); e != nil {
				return e
			}
		case an.TtMetaInfo:
			d1 := tlv.Decoder(field.Value)
			for _, field1 := range d1.Elements() {
				switch an.TlvType(field1.Type) {
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
		case an.TtSignatureInfo:
		case an.TtSignatureValue:
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
	sigType, _ := tlv.Encode(tlv.MakeElementNNI(an.TtSignatureType, 0))
	dataFakeSig, _ = tlv.Encode(
		tlv.MakeElement(an.TtSignatureInfo, sigType),
		tlv.MakeElement(an.TtSignatureValue, make([]byte, sha256.Size)),
	)

}
