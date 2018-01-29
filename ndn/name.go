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

type Name struct {
	c C.Name
}

// Decode a name.
func (d *TlvDecoder) ReadName() (n Name, e error) {
	res := C.DecodeName(d.getPtr(), &n.c)
	if res != C.NdnError_OK {
		return Name{}, NdnError(res)
	}
	return n, nil
}

func (n *Name) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(&n.c)
}

// Get number of name components.
func (n *Name) Len() int {
	return int(n.c.nComps)
}

// Test whether the name has an implicit digest.
func (n *Name) HasDigest() bool {
	return bool(n.c.hasDigestComp)
}

// Get component as TlvElement.
// i: zero-based index; count from the end if negative.
func (n *Name) GetCompAsElement(i int) (ele TlvElement) {
	if i < 0 {
		i += int(n.Len())
	}
	C.Name_GetComp(&n.c, C.uint16_t(i), &ele.c)
	return ele
}

// Compute hash for prefix with i component.
func (n *Name) ComputePrefixHash(i int) uint64 {
	return uint64(C.Name_ComputePrefixHash(&n.c, C.uint16_t(i)))
}

type NameCompareResult int

const (
	NAMECMP_LT      NameCompareResult = -2 // n is less than, but not a prefix of n2
	NAMECMP_LPREFIX                   = -1 // n is a prefix of n2
	NAMECMP_EQUAL                     = 0  // n and n2 are equal
	NAMECMP_RPREFIX                   = 1  // n2 is a prefix of n
	NAMECMP_GT                        = 2  // n2 is less than, but not a prefix of n
)

// Compare two names for <, ==, >, and prefix relations.
func (n *Name) Compare(n2 Name) NameCompareResult {
	return NameCompareResult(C.Name_Compare(&n.c, &n2.c))
}

func (n Name) String() string {
	if n.Len() == 0 {
		return "/"
	}

	var sb bytes.Buffer
	for i := 0; i < n.Len(); i++ {
		fmt.Fprint(&sb, "/")
		comp := n.GetCompAsElement(i)
		printNameComponent(&sb, &comp)
	}
	return sb.String()
}

func printNameComponent(w io.Writer, comp *TlvElement) (n int, e error) {
	switch comp.GetType() {
	case TT_ImplicitSha256DigestComponent:
		return printDigestComponent(w, comp)
	case TT_GenericNameComponent:
	default:
		n, e = fmt.Fprintf(w, "%v=", comp.GetType())
		if e != nil {
			return
		}
	}

	n2 := 0
	nNonPeriods := 0
	for _, b := range comp.GetValue() {
		if ('A' <= b && b <= 'Z') || ('a' <= b && b <= 'z') || ('0' <= b && b <= '9') ||
			b == '+' || b == '.' || b == '_' || b == '-' {
			n2, e = fmt.Fprint(w, string(b))
		} else {
			n2, e = fmt.Fprintf(w, "%%%02X", b)
		}
		n += n2
		if e != nil {
			return
		}
		if b != '.' {
			nNonPeriods++
		}
	}

	if nNonPeriods == 0 {
		n2, e = fmt.Fprint(w, "...")
		n += n2
	}
	return
}

func printDigestComponent(w io.Writer, comp *TlvElement) (n int, e error) {
	n, e = fmt.Fprint(w, "sha256digest=")
	if e != nil {
		return
	}

	n2 := 0
	for _, b := range comp.GetValue() {
		n2, e = fmt.Fprintf(w, "%02x", b)
		n += n2
		if e != nil {
			return
		}
	}
	return
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
