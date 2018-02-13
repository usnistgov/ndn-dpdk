package ndn

/*
#include "name.h"
*/
import "C"
import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"unsafe"
)

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
	if res != C.NdnError_OK {
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
	return (*C.uint8_t)(n.b.GetPtr())
}

// Get number of name components.
func (n *Name) Len() int {
	return int(n.p.nComps)
}

// Get TLV-LENGTH of Name element.
func (n *Name) Size() int {
	return int(n.p.nOctets)
}

// Test whether the name ends with an implicit digest.
func (n *Name) HasDigestComp() bool {
	return bool(n.p.hasDigestComp)
}

// Get i-th name component TLV.
func (n *Name) GetComp(i int) NameComponent {
	start := C.PName_GetCompStart(n.p, n.getValuePtr(), C.uint16_t(i))
	end := C.PName_GetCompEnd(n.p, n.getValuePtr(), C.uint16_t(i))
	return NameComponent(n.b[start:end])
}

// Get all name component TLVs.
func (n *Name) ListComps() []NameComponent {
	comps := make([]NameComponent, n.Len())
	for i := range comps {
		comps[i] = n.GetComp(i)
	}
	return comps
}

// Compute hash for prefix with i components.
func (n *Name) ComputePrefixHash(i int) uint64 {
	return uint64(C.PName_ComputePrefixHash(n.p, n.getValuePtr(), C.uint16_t(i)))
}

// Compute hash for prefix with i components.
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

// Print as URI.
// Implements io.WriterTo.
func (n *Name) WriteTo(w io.Writer) (nn int64, e error) {
	if n.Len() == 0 {
		n2, e := fmt.Fprint(w, "/")
		return int64(n2), e
	}

	for _, comp := range n.ListComps() {
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

// Encode name from URI.
// Limitation: this function does not recognize typed components,
// and cannot detect certain invalid names.
func EncodeNameFromUri(uri string) (TlvBytes, error) {
	buf, e := EncodeNameComponentsFromUri(uri)
	if e != nil {
		return nil, e
	}
	return append(EncodeTlvTypeLength(TT_Name, len(buf)), buf...), nil
}

// Parse name from URI and encode components only.
// Limitation: this function does not recognize typed components,
// and cannot detect certain invalid names.
func EncodeNameComponentsFromUri(uri string) (TlvBytes, error) {
	uri = strings.TrimPrefix(uri, "ndn:")
	uri = strings.TrimPrefix(uri, "/")

	var buf bytes.Buffer
	if uri != "" {
		for i, token := range strings.Split(uri, "/") {
			comp, e := encodeNameComponentFromUri(token)
			if e != nil {
				return nil, fmt.Errorf("component %d '%s': %v", i, token, e)
			}
			buf.Write(comp)
		}
	}

	if buf.Len() == 0 {
		return oneTlvByte[:0], nil
	}
	return buf.Bytes(), nil
}

func encodeNameComponentFromUri(token string) (TlvBytes, error) {
	if strings.Contains(token, "=") {
		return nil, fmt.Errorf("typed component is not supported")
	}

	var buf bytes.Buffer
	if strings.TrimLeft(token, ".") == "" {
		if len(token) < 3 {
			return nil, fmt.Errorf("invalid URI component of less than three periods")
		}
		buf.WriteString(token[3:])
	} else {
		for i := 0; i < len(token); i++ {
			ch := token[i]
			if ch == '%' && i+2 < len(token) {
				b, e := hex.DecodeString(token[i+1 : i+3])
				if e != nil {
					return nil, fmt.Errorf("hex error near position %d: %v", i, e)
				}
				buf.Write(b)
				i += 2
			} else {
				buf.WriteByte(ch)
			}
		}
	}

	return append(EncodeTlvTypeLength(TT_GenericNameComponent, buf.Len()), buf.Bytes()...), nil
}

func EncodeNameComponentFromNumber(tlvType TlvType, v interface{}) TlvBytes {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, v)
	return append(EncodeTlvTypeLength(tlvType, buf.Len()), buf.Bytes()...)
}
