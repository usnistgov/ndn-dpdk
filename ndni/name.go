package ndni

/*
#include "../csrc/ndn/name.h"
*/
import "C"
import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

const NAME_MAX_LENGTH = C.NAME_MAX_LENGTH

// Name element.
type Name struct {
	b TlvBytes
	p *C.PName
}

// Parse name from TLV-VALUE of Name element.
func NewName(b TlvBytes) (n *Name, e error) {
	n = new(Name)
	n.b = b
	n.p = new(C.PName)
	res := C.PName_Parse(n.p, C.uint32_t(len(b)), n.getValuePtr())
	if res != C.NdnErrOK {
		return nil, NdnError(res)
	}
	return n, nil
}

func (n *Name) copyFromC(c *C.Name) {
	n.b = TlvBytes(C.GoBytes(unsafe.Pointer(c.v), C.int(c.p.nOctets)))
	n.p = new(C.PName)
	*n.p = c.p
}

func (n *Name) getValuePtr() *C.uint8_t {
	if n == nil {
		return nil
	}
	return (*C.uint8_t)(n.b.GetPtr())
}

// Get number of name components.
func (n *Name) Len() int {
	return int(n.p.nComps)
}

// Get TLV-LENGTH of Name element.
func (n *Name) Size() int {
	if n == nil {
		return 0
	}
	return int(n.p.nOctets)
}

// Get *C.PName pointer.
func (n *Name) GetPNamePtr() unsafe.Pointer {
	return unsafe.Pointer(n.p)
}

// Get TLV-VALUE of Name element.
func (n *Name) GetValue() TlvBytes {
	return n.b
}

// Encode Name element to TlvBytes.
func (n *Name) Encode() TlvBytes {
	return EncodeTlv(an.TtName, n.b)
}

// Copy to C.LName.
func (n *Name) CopyToLName(lname, buffer unsafe.Pointer, sizeofBuffer uintptr) error {
	sz := uintptr(n.Size())
	if sizeofBuffer < sz {
		return fmt.Errorf("buffer too short, need %d", sz)
	}

	nameV := unsafe.Pointer(n.getValuePtr())
	for i := uintptr(0); i < sz; i++ {
		src := unsafe.Pointer(uintptr(nameV) + i)
		dst := unsafe.Pointer(uintptr(buffer) + i)
		*(*uint8)(dst) = *(*uint8)(src)
	}

	lnameC := (*C.LName)(lname)
	lnameC.length = C.uint16_t(sz)
	lnameC.value = (*C.uint8_t)(buffer)
	return nil
}

// Test whether the name ends with an implicit digest.
func (n *Name) HasDigestComp() bool {
	return bool(n.p.hasDigestComp)
}

// Get i-th name component TLV.
func (n *Name) GetComp(i int) NameComponent {
	begin := C.PName_GetCompBegin(n.p, n.getValuePtr(), C.uint16_t(i))
	end := C.PName_GetCompEnd(n.p, n.getValuePtr(), C.uint16_t(i))
	return NameComponent(n.b[begin:end])
}

// Get a name prefix of certain length.
func (n *Name) GetPrefix(prefixLen int) (prefix *Name) {
	if prefixLen <= 0 {
		prefix, _ = NewName(nil)
	} else {
		end := C.PName_GetCompEnd(n.p, n.getValuePtr(), C.uint16_t(prefixLen-1))
		prefix, _ = NewName(n.GetValue()[:end])
	}
	return prefix
}

// Compute hash for prefix with i components.
func (n *Name) ComputePrefixHash(i int) uint64 {
	return uint64(C.PName_ComputePrefixHash(n.p, n.getValuePtr(), C.uint16_t(i)))
}

// Compute hash for all components.
func (n *Name) ComputeHash() uint64 {
	return uint64(C.PName_ComputeHash(n.p, n.getValuePtr()))
}

// Indicate the result of name comparison.
type NameCompareResult int

const (
	NAMECMP_LT      NameCompareResult = -2 // lhs is less than, but not a prefix of rhs
	NAMECMP_LPREFIX NameCompareResult = -1 // lhs is a prefix of rhs
	NAMECMP_EQUAL   NameCompareResult = 0  // lhs and rhs are equal
	NAMECMP_RPREFIX NameCompareResult = 1  // rhs is a prefix of lhs
	NAMECMP_GT      NameCompareResult = 2  // rhs is less than, but not a prefix of lhs
)

// Compare two names for <, ==, >, and prefix relations.
func (n *Name) Compare(r *Name) NameCompareResult {
	lhs := C.LName{value: n.getValuePtr(), length: n.p.nOctets}
	rhs := C.LName{value: r.getValuePtr(), length: r.p.nOctets}
	return NameCompareResult(C.LName_Compare(lhs, rhs))
}

// Determine if two names are equal.
func (n *Name) Equal(r *Name) bool {
	return n.Compare(r) == NAMECMP_EQUAL
}

// Print as URI.
// Implements io.WriterTo.
func (n *Name) WriteTo(w io.Writer) (nn int64, e error) {
	if n.Len() == 0 {
		n2, e := fmt.Fprint(w, "/")
		return int64(n2), e
	}
	for i := 0; i < n.Len(); i++ {
		comp := n.GetComp(i)
		if n2, e := fmt.Fprint(w, "/"); e != nil {
			return nn, e
		} else {
			nn += int64(n2)
		}
		if n2, e := comp.WriteTo(w); e != nil {
			return nn, e
		} else {
			nn += int64(n2)
		}
	}
	return nn, nil
}

// Convert to URI.
func (n *Name) String() string {
	var sb bytes.Buffer
	n.WriteTo(&sb)
	return sb.String()
}

// Parse name from URI.
func ParseName(uri string) (n *Name, e error) {
	uri = strings.TrimPrefix(uri, "ndn:")
	uri = strings.TrimPrefix(uri, "/")

	var buf bytes.Buffer
	if uri != "" {
		for i, token := range strings.Split(uri, "/") {
			if comp, e := ParseNameComponent(token); e != nil {
				return nil, fmt.Errorf("component %d '%s': %v", i, token, e)
			} else {
				buf.Write(comp)
			}
		}
	}

	if buf.Len() == 0 {
		return NewName(oneTlvByte[:0])
	}
	return NewName(buf.Bytes())
}

// Parse name from URI.
// Panics if parsing fails.
func MustParseName(uri string) *Name {
	n, e := ParseName(uri)
	if e != nil {
		panic(e)
	}
	return n
}

func (n *Name) MarshalText() (text []byte, e error) {
	return []byte(n.String()), nil
}

func (n *Name) UnmarshalText(text []byte) error {
	n2, e := ParseName(string(text))
	if e == nil {
		*n = *n2
	}
	return e
}
