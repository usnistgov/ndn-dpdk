package ndn

/*
#include "tlv-encoder.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

type TlvEncoder struct {
	c *C.TlvEncoder
}

func NewTlvEncoder(pkt dpdk.Packet) (encoder TlvEncoder) {
	encoder.c = C.MakeTlvEncoder((*C.struct_rte_mbuf)(pkt.GetPtr()))
	return encoder
}

// TLV bytes in Go memory.
type TlvBytes []byte

var oneTlvByte = make(TlvBytes, 1)

// Get C pointer to the first octet.
func (tb TlvBytes) GetPtr() unsafe.Pointer {
	if len(tb) == 0 {
		return unsafe.Pointer(&oneTlvByte[0])
	}
	return unsafe.Pointer(&tb[0])
}

// Count how many elements are present in TlvBytes.
// Return the number of elements, or -1 if incomplete.
func (tb TlvBytes) CountElements() (n int) {
	b := []byte(tb)
	for _, size, ok := DecodeVarNum(b); ok; _, size, ok = DecodeVarNum(b) { // read TLV-TYPE
		b = b[size:]
		if length, size, ok := DecodeVarNum(b); !ok || len(b) < size+int(length) { // read TLV-LENGTH
			return -1
		} else {
			b = b[size+int(length):]
		}
		n++
	}
	if len(b) > 0 {
		return -1
	}
	return n
}

func EncodeTlvTypeLength(tlvType TlvType, tlvLength int) TlvBytes {
	return append(EncodeVarNum(uint64(tlvType)), EncodeVarNum(uint64(tlvLength))...)
}

func EncodeVarNum(n uint64) TlvBytes {
	buf := make([]byte, int(C.SizeofVarNum(C.uint64_t(n))))
	C.EncodeVarNum((*C.uint8_t)(unsafe.Pointer(&buf[0])), C.uint64_t(n))
	return buf
}
