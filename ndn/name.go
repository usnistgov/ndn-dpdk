package ndn

import (
	"strings"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Name represents a name.
// The zero Name has zero components.
type Name []NameComponent

// Length returns TLV-LENGTH.
// Use len(name) to get number of components.
func (name Name) Length() int {
	sum := 0
	for _, comp := range name {
		sum += comp.Size()
	}
	return sum
}

// Compare returns negative when name<other, zero when name==other, positive when name>other.
func (name Name) Compare(other Name) int {
	if d := name.compareCommonPrefix(other); d != 0 {
		return d
	}
	return len(name) - len(other)
}

// IsPrefixOf returns true if this name is a prefix of other name.
func (name Name) IsPrefixOf(other Name) bool {
	if d := name.compareCommonPrefix(other); d != 0 {
		return false
	}
	return len(name) <= len(other)
}

func (name Name) compareCommonPrefix(other Name) int {
	commonPrefixLen := len(name)
	if commonPrefixLen > len(other) {
		commonPrefixLen = len(other)
	}
	for i := 0; i < commonPrefixLen; i++ {
		if d := name[i].Compare(other[i]); d != 0 {
			return d
		}
	}
	return 0
}

// MarshalBinary encodes TLV-VALUE of this name.
func (name Name) MarshalBinary() (wire []byte, e error) {
	return tlv.EncodeValue(([]NameComponent)(name))
}

// UnmarshalBinary decodes TLV-VALUE from wire format.
func (name *Name) UnmarshalBinary(wire []byte) (e error) {
	d := tlv.Decoder(wire)
	*name = make(Name, 0)
	for !d.EOF() {
		var comp NameComponent
		if e := d.Unmarshal(&comp); e != nil {
			return e
		}
		*name = append(*name, comp)
	}
	return nil
}

// MarshalTlv encodes this name.
func (name Name) MarshalTlv() (wire []byte, e error) {
	return tlv.EncodeElement(an.TtName, ([]NameComponent)(name))
}

// UnmarshalTlv decodes from wire format.
func (name *Name) UnmarshalTlv(wire []byte) (rest []byte, e error) {
	element, rest, e := tlv.DecodeFirstExpect(an.TtName, wire)
	if e != nil {
		return nil, e
	}

	e = name.UnmarshalBinary(element.Value)
	return rest, e
}

// MarshalText implements encoding.TextMarshaler interface.
func (name Name) MarshalText() (text []byte, e error) {
	return []byte(name.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler interface.
func (name *Name) UnmarshalText(text []byte) error {
	*name = ParseName(string(text))
	return nil
}

// String returns URI representation of this name.
func (name Name) String() string {
	if len(name) == 0 {
		return "/"
	}
	var w strings.Builder
	for _, comp := range name {
		w.WriteByte('/')
		comp.writeStringTo(&w)
	}
	return w.String()
}

// ParseName parses URI representation of name.
// It uses best effort and can accept any input.
func ParseName(input string) (name Name) {
	input = strings.TrimPrefix(input, "ndn:")
	for _, token := range strings.Split(input, "/") {
		if token == "" {
			continue
		}
		comp := ParseNameComponent(token)
		name = append(name, comp)
	}
	return name
}
