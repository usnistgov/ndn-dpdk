package ndn

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk
#cgo LDFLAGS: -L../build-c -lndn-traffic-dpdk-dpdk

#include "tlv-decoder.h"
*/
import "C"
import (
	"ndn-traffic-dpdk/dpdk"
)

type TlvDecoder struct {
	it dpdk.PacketIterator
}

func NewTlvDecoder(pkt dpdk.Packet) TlvDecoder {
	return TlvDecoder{dpdk.NewPacketIterator(pkt)}
}

func (d *TlvDecoder) getPtr() *C.MbufLoc {
	return (*C.MbufLoc)(d.it.GetPtr())
}

// Decode a TLV-TYPE or TLV-LENGTH number.
func (d *TlvDecoder) ReadVarNum() (v uint64, length uint, e error) {
	var lengthC C.size_t
	res := C.DecodeVarNum(d.getPtr(), (*C.uint64_t)(&v), &lengthC)
	if res != C.NdnError_OK {
		return 0, 0, NdnError(res)
	}
	return v, uint(lengthC), nil
}

type TlvElement struct {
	c C.TlvElement
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

// Decode a TLV element.
func (d *TlvDecoder) ReadTlvElement() (ele TlvElement, length uint, e error) {
	var lengthC C.size_t
	res := C.DecodeTlvElement(d.getPtr(), &ele.c, &lengthC)
	if res != C.NdnError_OK {
		return TlvElement{}, 0, NdnError(res)
	}
	return ele, uint(lengthC), nil
}
