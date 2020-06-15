package ndn

/*
#include "../csrc/ndn/tlv-encoder.h"
#include "../csrc/ndn/tlv-varnum.h"
*/
import "C"
import (
	"bytes"
	"encoding/hex"
	"strings"
	"unsafe"
)

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

// Compare equality.
func (tb TlvBytes) Equal(other TlvBytes) bool {
	return bytes.Equal([]byte(tb), []byte(other))
}

// Decode a TLV-TYPE or TLV-LENGTH number.
// Return the number and remaining bytes, or (0, nil) if failed.
func (tb TlvBytes) DecodeVarNum() (v uint32, tail TlvBytes) {
	if len(tb) < 1 {
		return 0, nil
	}
	res := C.DecodeVarNum((*C.uint8_t)(tb.GetPtr()), C.uint32_t(len(tb)), (*C.uint32_t)(&v))
	if res <= 0 {
		return 0, nil
	}
	return v, tb[res:]
}

// Extract the first element from TlvBytes.
// Return the first element or nil if not found, and any remaining bytes.
func (tb TlvBytes) ExtractElement() (element TlvBytes, tail TlvBytes) {
	if _, tail = tb.DecodeVarNum(); tail == nil {
		return nil, tb
	}
	var length uint32
	if length, tail = tail.DecodeVarNum(); tail == nil || len(tail) < int(length) {
		return nil, tb
	}
	tail = tail[int(length):]
	element = tb[:len(tb)-len(tail)]
	return element, tail
}

func (tb TlvBytes) Join(tail ...TlvBytes) TlvBytes {
	a := [][]byte{tb}
	for _, part := range tail {
		a = append(a, ([]byte)(part))
	}
	return TlvBytes(bytes.Join(a, nil))
}

func (tb TlvBytes) String() string {
	return strings.ToUpper(hex.EncodeToString(([]byte)(tb)))
}

// Encode TLV from TLV-TYPE and TLV-VALUE chunks.
func EncodeTlv(tlvType TlvType, value ...TlvBytes) TlvBytes {
	length := 0
	for _, v := range value {
		length += len(v)
	}

	sizeofT := C.SizeofVarNum(C.uint32_t(tlvType))
	sizeofL := C.SizeofVarNum(C.uint32_t(length))
	tl := make(TlvBytes, int(sizeofT+sizeofL))
	C.EncodeVarNum((*C.uint8_t)(unsafe.Pointer(&tl[0])), C.uint32_t(tlvType))
	C.EncodeVarNum((*C.uint8_t)(unsafe.Pointer(&tl[sizeofT])), C.uint32_t(length))

	return tl.Join(value...)
}
