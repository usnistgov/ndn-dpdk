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
func (buf TlvBytes) GetPtr() unsafe.Pointer {
	if len(buf) == 0 {
		return unsafe.Pointer(&oneTlvByte[0])
	}
	return unsafe.Pointer(&buf[0])
}

func EncodeTlvTypeLength(tlvType TlvType, tlvLength int) TlvBytes {
	return append(EncodeVarNum(uint64(tlvType)), EncodeVarNum(uint64(tlvLength))...)
}

func EncodeVarNum(n uint64) TlvBytes {
	buf := make([]byte, int(C.SizeofVarNum(C.uint64_t(n))))
	C.EncodeVarNum((*C.uint8_t)(unsafe.Pointer(&buf[0])), C.uint64_t(n))
	return buf
}
