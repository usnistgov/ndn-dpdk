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

// Get returns i-th component.
// If negative, count from the end.
// If out-of-range, return invalid NameComponent.
func (name Name) Get(i int) NameComponent {
	if i < 0 {
		i += len(name)
	}
	if i < 0 || i >= len(name) {
		return NameComponent{}
	}
	return name[i]
}

// Slice returns a sub name between i-th (inclusive) and j-th (exclusive) components.
// j is optional; the default is toward the end.
// If negative, count from the end.
// If out-of-range, return empty name.
func (name Name) Slice(i int, j ...int) Name {
	if i < 0 {
		i += len(name)
	}

	switch len(j) {
	case 0:
		if i < 0 || i >= len(name) {
			return Name{}
		}
		return name[i:]
	case 1:
		jj := j[0]
		if jj < 0 {
			jj += len(name)
		}
		if i < 0 || i >= jj || jj > len(name) {
			return Name{}
		}
		return name[i:jj]
	default:
		panic("name.Slice takes 1 or 2 arguments")
	}
}

// GetPrefix returns a prefix of i components.
// If negative, count from the end.
func (name Name) GetPrefix(i int) Name {
	return name.Slice(0, i)
}

// Equal determins whether two names are the same.
func (name Name) Equal(other Name) bool {
	return name.Compare(other) == 0
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

// MarshalTlv encodes this name.
func (name Name) MarshalTlv() (typ uint32, value []byte, e error) {
	return tlv.EncodeTlv(an.TtName, ([]NameComponent)(name))
}

// MarshalBinary encodes TLV-VALUE of this name.
func (name Name) MarshalBinary() (value []byte, e error) {
	return tlv.Encode(([]NameComponent)(name))
}

// UnmarshalBinary decodes TLV-VALUE from wire format.
func (name *Name) UnmarshalBinary(wire []byte) error {
	*name = make(Name, 0)
	d := tlv.Decoder(wire)
	for _, element := range d.Elements() {
		var comp NameComponent
		if e := element.Unmarshal(&comp); e != nil {
			return e
		}
		*name = append(*name, comp)
	}
	return d.ErrUnlessEOF()
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
