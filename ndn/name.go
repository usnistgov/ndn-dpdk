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
)

type Name struct {
	b TlvBytes
	p *C.PName
}

func NewName(b TlvBytes) (n *Name, e error) {
	n = new(Name)
	n.p = new(C.PName)
	res := C.PName_Parse(n.p, C.uint32_t(len(b)), (*C.uint8_t)(b.GetPtr()))
	if res != C.NdnError_OK {
		return nil, NdnError(res)
	}
	n.b = b
	return n, nil
}

func (n *Name) Len() int {
	return int(n.p.nComps)
}

func (n *Name) HasDigestComp() bool {
	return bool(n.p.hasDigestComp)
}

func (n *Name) GetComp(i int) TlvBytes {
	start := C.PName_GetCompStart(n.p, (*C.uint8_t)(n.b.GetPtr()), C.uint16_t(i))
	end := C.PName_GetCompStart(n.p, (*C.uint8_t)(n.b.GetPtr()), C.uint16_t(i))
	return n.b[start:end]
}

type NameCompareResult int

const (
	NAMECMP_LT      NameCompareResult = -2 // n is less than, but not a prefix of n2
	NAMECMP_LPREFIX                   = -1 // n is a prefix of n2
	NAMECMP_EQUAL                     = 0  // n and n2 are equal
	NAMECMP_RPREFIX                   = 1  // n2 is a prefix of n
	NAMECMP_GT                        = 2  // n2 is less than, but not a prefix of n
)

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
