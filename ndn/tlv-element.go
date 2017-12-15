package ndn

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk
#cgo LDFLAGS: -L../build-c -lndn-traffic-dpdk-dpdk

#include "tlv-element.h"
*/
import "C"
import "fmt"

type TlvElement struct {
	c C.TlvElement
}

// Decode a TLV element.
func (d *TlvDecoder) ReadTlvElement() (ele TlvElement, e error) {
	res := C.DecodeTlvElement(d.getPtr(), &ele.c)
	if res != C.NdnError_OK {
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
func (ele *TlvElement) GetValue() []byte {
	var d TlvDecoder
	C.TlvElement_MakeValueDecoder(&ele.c, d.getPtr())

	b := make([]byte, ele.GetLength())
	d.it.Read(b) // will always succeed on valid TLV
	return b
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
