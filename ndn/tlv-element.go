package ndn

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk
#cgo LDFLAGS: -L../build-c -lndn-traffic-dpdk-dpdk

#include "tlv-element.h"
*/
import "C"

type TlvElement struct {
	c C.TlvElement
}

// Decode a TLV element.
func (d *TlvDecoder) ReadTlvElement() (ele TlvElement, length uint, e error) {
	var lengthC C.size_t
	res := C.DecodeTlvElement(d.getPtr(), &ele.c, &lengthC)
	if res != C.NdnError_OK {
		return TlvElement{}, 0, NdnError(res)
	}
	return ele, uint(lengthC), nil
}

// Get total length.
func (ele *TlvElement) Len() uint {
	return uint(ele.c.size)
}

// Get TLV-TYPE.
func (ele *TlvElement) GetType() uint64 {
	return uint64(ele.c._type)
}

// Get TLV-LENGTH.
func (ele *TlvElement) GetLength() uint {
	return uint(ele.c.length)
}
