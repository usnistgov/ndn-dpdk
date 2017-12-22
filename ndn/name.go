package ndn

/*
#include "name.h"
*/
import "C"
import (
	"bytes"
	"fmt"
	"io"
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

// Get number of name components.
func (n *Name) Len() int {
	return int(n.c.nComps)
}

// Test whether the name has an implicit digest.
func (n *Name) HasDigest() bool {
	return bool(C.Name_HasDigest(&n.c))
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
