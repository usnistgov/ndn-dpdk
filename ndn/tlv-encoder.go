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

func EncodeTlvTypeLength(tlvType TlvType, tlvLength int) []byte {
	return append(EncodeVarNum(uint64(tlvType)), EncodeVarNum(uint64(tlvLength))...)
}

func EncodeVarNum(n uint64) []byte {
	buf := make([]byte, int(C.SizeofVarNum(C.uint64_t(n))))
	C.EncodeVarNum((*C.uint8_t)(unsafe.Pointer(&buf[0])), C.uint64_t(n))
	return buf
}
