package ndn

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// NameComponent represents a name component.
// The zero NameComponent is invalid.
type NameComponent struct {
	tlv.Element
}

// MakeNameComponent constructs a NameComponent from TLV-TYPE and TLV-VALUE.
func MakeNameComponent(typ interface{}, value []byte) (comp NameComponent) {
	comp.Element = tlv.MakeElement(typ, value)
	return comp
}

// Valid checks whether this component has a valid TLV-TYPE.
func (comp NameComponent) Valid() bool {
	return isValidNameComponentType(comp.Type)
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

// MarshalTlv encodes this component.
func (comp NameComponent) MarshalTlv() (typ uint32, value []byte, e error) {
	if !comp.Valid() {
		return 0, nil, ErrComponentType
	}
	return comp.Element.MarshalTlv()
}

// UnmarshalTlv decodes from wire format.
func (comp *NameComponent) UnmarshalTlv(typ uint32, value []byte) error {
	if e := comp.Element.UnmarshalTlv(typ, value); e != nil {
		return e
	}
	if !comp.Valid() {
		return ErrComponentType
	}
	return nil
}

// String returns URI representation of this component.
func (comp NameComponent) String() string {
	var w strings.Builder
	comp.writeStringTo(&w)
	return w.String()
}

func (comp NameComponent) writeStringTo(w *strings.Builder) {
	w.WriteString(strconv.Itoa(int(comp.Type)))
	w.WriteByte('=')

	nNonPeriods := 0
	for _, b := range comp.Value {
		if bytes.IndexByte(unescapedChars, b) >= 0 {
			w.WriteByte(b)
		} else {
			w.WriteByte('%')
			w.WriteByte(hexChars[b>>4])
			w.WriteByte(hexChars[b&0x0F])
		}
		if b != '.' {
			nNonPeriods++
		}
	}

	if nNonPeriods == 0 {
		w.WriteString("...")
	}
}

// ParseNameComponent parses URI representation of name component.
// It uses best effort and can accept any input.
func ParseNameComponent(input string) (comp NameComponent) {
	comp.Type = uint32(an.TtGenericNameComponent)
	pos := strings.IndexByte(input, '=')
	if pos >= 0 {
		typ, e := strconv.ParseUint(input[:pos], 10, 32)
		typ32 := uint32(typ)
		if e == nil && isValidNameComponentType(typ32) {
			comp.Type = typ32
			pos++
		} else {
			pos = 0
		}
	} else {
		pos = 0
	}

	if len(strings.TrimRight(input, ".")) == pos && len(input) >= 3 {
		comp.Value = []byte(input)[pos+3:]
		return comp
	}

	var value bytes.Buffer
	for i := pos; i < len(input); {
		ch := input[i]
		if ch == '%' && i+2 < len(input) {
			b, e := strconv.ParseUint(input[i+1:i+3], 16, 8)
			if e == nil {
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

func isValidNameComponentType(typ uint32) bool {
	return typ >= 1 && typ <= 65535
}

var unescapedChars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~")
var hexChars = []byte("0123456789ABCDEF")
