package ndn

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

var (
	unescapedChars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~")
	hexChars       = []byte("0123456789ABCDEF")
)

func isValidNameComponentType(typ uint32) bool {
	return typ >= 1 && typ <= 65535
}

// NameComponent represents a name component.
// Zero value is invalid.
type NameComponent struct {
	tlv.Element
}

var (
	_ tlv.Fielder     = NameComponent{}
	_ tlv.Unmarshaler = (*NameComponent)(nil)
)

// Valid checks whether this component has a valid TLV-TYPE.
func (comp NameComponent) Valid() bool {
	return isValidNameComponentType(comp.Type)
}

// Equal determines whether two components are the same.
func (comp NameComponent) Equal(other NameComponent) bool {
	return comp.Compare(other) == 0
}

// Compare returns negative when comp<other, zero when comp==other, positive when comp>other.
func (comp NameComponent) Compare(other NameComponent) int {
	if d := int(comp.Type) - int(other.Type); d != 0 {
		return d
	}
	if d := comp.Length() - other.Length(); d != 0 {
		return d
	}
	return bytes.Compare(comp.Value, other.Value)
}

// Field implements tlv.Fielder interface.
func (comp NameComponent) Field() tlv.Field {
	if !comp.Valid() {
		return tlv.FieldError(ErrComponentType)
	}
	return comp.Element.Field()
}

// UnmarshalTLV decodes from wire format.
func (comp *NameComponent) UnmarshalTLV(typ uint32, value []byte) error {
	if e := comp.Element.UnmarshalTLV(typ, value); e != nil {
		return e
	}
	if !comp.Valid() {
		return ErrComponentType
	}
	return nil
}

// String returns URI representation of this component.
func (comp NameComponent) String() string {
	return string(comp.appendStringTo(make([]byte, 0, 6+3*comp.Length())))
}

func (comp NameComponent) appendStringTo(b []byte) []byte {
	b = strconv.AppendUint(b, uint64(comp.Type), 10)
	b = append(b, '=')

	allPeriods := true
	for _, ch := range comp.Value {
		if bytes.IndexByte(unescapedChars, ch) >= 0 {
			b = append(b, ch)
		} else {
			b = append(b, '%', hexChars[ch>>4], hexChars[ch&0x0F])
		}
		allPeriods = allPeriods && (ch == '.')
	}

	if allPeriods {
		b = append(b, '.', '.', '.')
	}
	return b
}

// MakeNameComponent constructs a NameComponent from TLV-TYPE and TLV-VALUE.
func MakeNameComponent(typ uint32, value []byte) (comp NameComponent) {
	comp.Element = tlv.Element{
		Type:  typ,
		Value: value,
	}
	return comp
}

// NameComponentFrom constructs a NameComponent from TLV-TYPE and tlv.Fielder as TLV-VALUE.
// If value encodes to an error, returns an invalid NameComponent.
//
// To create a name component with NonNegativeInteger as commonly used in naming conventions:
//  NameComponentFrom(an.VersionNameComponent, tlv.NNI(1))
func NameComponentFrom(typ uint32, value tlv.Fielder) NameComponent {
	v, e := value.Field().Encode(nil)
	if e != nil {
		return NameComponent{}
	}
	return MakeNameComponent(typ, v)
}

// ParseNameComponent parses canonical URI representation of name component.
// It uses best effort and can accept any input.
func ParseNameComponent(input string) (comp NameComponent) {
	comp.Type = an.TtGenericNameComponent
	if typS, valS, hasTyp := strings.Cut(input, "="); hasTyp {
		typ64, e := strconv.ParseUint(typS, 10, 32)
		typ32 := uint32(typ64)
		if e == nil && isValidNameComponentType(typ32) {
			comp.Type = typ32
			input = valS
		}
	}

	if len(strings.TrimRight(input, ".")) == 0 && len(input) >= 3 {
		comp.Value = []byte(input)[3:]
		return comp
	}

	var value bytes.Buffer
	value.Grow(len(input))
	for i := 0; i < len(input); {
		ch := input[i]
		if ch == '%' && i+2 < len(input) {
			if b, e := strconv.ParseUint(input[i+1:i+3], 16, 8); e == nil {
				value.WriteByte(byte(b))
				i += 3
				continue
			}
		}
		value.WriteByte(ch)
		i++
	}
	comp.Value = value.Bytes()
	return comp
}
