package ndn

/*
#include "tlv-element.h"
*/
import "C"
import (
	"fmt"
)

type TlvElement struct {
	c C.TlvElement
}

// Decode a TLV element.
func (d *TlvDecodePos) ReadTlvElement() (ele TlvElement, e error) {
	if res := C.TlvElement_Decode(&ele.c, d.getPtr(), C.TT_Invalid); res != C.NdnError_OK {
		return TlvElement{}, NdnError(res)
	}
	return ele, nil
}

// Get total length.
func (ele *TlvElement) Len() int {
	return int(ele.c.size)
}

// Get TLV-TYPE.
func (ele *TlvElement) GetType() TlvType {
	return TlvType(ele.c._type)
}

// Get TLV-LENGTH.
func (ele *TlvElement) GetLength() int {
	return int(ele.c.length)
}

// Get TLV-VALUE.
func (ele *TlvElement) GetValue() (v TlvBytes) {
	var d TlvDecodePos
	C.TlvElement_MakeValueDecoder(&ele.c, d.getPtr())

	v = make(TlvBytes, ele.GetLength())
	d.it.Read(([]byte)(v)) // will always succeed on valid TLV
	return v
}

// Interpret TLV-VALUE as NonNegativeInteger.
func (ele *TlvElement) ReadNonNegativeInteger() (n uint64, ok bool) {
	var v C.uint64_t
	res := C.TlvElement_ReadNonNegativeInteger(&ele.c, &v)
	return uint64(v), bool(res)
}

func (ele *TlvElement) String() string {
	return fmt.Sprintf("%v(%d) %v", ele.GetType(), ele.GetLength(), ele.GetValue())
}
