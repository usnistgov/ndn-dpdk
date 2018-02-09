package ndn

/*
#include "name1.h"
*/
import "C"
import (
	"bytes"
	"fmt"
	"unsafe"
)

type Name1 struct {
	c C.Name1
}

// Decode a name.
func (d *TlvDecodePos) ReadName1() (n Name1, e error) {
	res := C.DecodeName1(d.getPtr(), &n.c)
	if res != C.NdnError_OK {
		return Name1{}, NdnError(res)
	}
	return n, nil
}

func (n *Name1) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(&n.c)
}

// Get number of name components.
func (n *Name1) Len() int {
	return int(n.c.nComps)
}

// Get TLV-LENGTH of Name element.
func (n *Name1) Size() int {
	return int(n.c.nOctets)
}

// Test whether the name1.has an implicit digest.
func (n *Name1) HasDigest() bool {
	return bool(n.c.hasDigestComp)
}

// Get component as TlvElement.
// i: zero-based index; count from the end if negative.
func (n *Name1) GetCompAsElement(i int) (ele TlvElement) {
	if i < 0 {
		i += int(n.Len())
	}
	C.Name1_GetComp(&n.c, C.uint16_t(i), &ele.c)
	return ele
}

// Get size of prefix with i components.
func (n *Name1) GetPrefixSize(i int) int {
	return int(C.Name1_GetPrefixSize(&n.c, C.uint16_t(i)))
}

// Compute hash for prefix with i components.
func (n *Name1) ComputePrefixHash(i int) uint64 {
	return uint64(C.Name1_ComputePrefixHash(&n.c, C.uint16_t(i)))
}

// Compare two names for <, ==, >, and prefix relations.
func (n *Name1) Compare(n2 Name1) NameCompareResult {
	return NameCompareResult(C.Name1_Compare(&n.c, &n2.c))
}

func (n Name1) String() string {
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
