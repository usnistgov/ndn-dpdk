package ndn

/*
#include "tlv-decode.h"
#include "tlv-encoder.h"
*/
import "C"
import (
	"bytes"
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
func (tb TlvBytes) DecodeVarNum() (v uint64, tail TlvBytes) {
	if len(tb) < 1 {
		return 0, nil
	}
	res := C.ParseVarNum((*C.uint8_t)(tb.GetPtr()), C.uint32_t(len(tb)), (*C.uint64_t)(&v))
	if res == 0 {
		return 0, nil
	}
	return v, tb[res:]
}

// Count how many elements are present in TlvBytes.
// Return the number of elements, or -1 if incomplete.
func (tb TlvBytes) CountElements() (n int) {
	for len(tb) > 0 {
		var length uint64
		if _, tb = tb.DecodeVarNum(); tb == nil { // read TLV-TYPE
			return -1
		}
		if length, tb = tb.DecodeVarNum(); tb == nil || len(tb) < int(length) { // read TLV-LENGTH
			return -1
		}
		tb = tb[length:]
		n++
	}
	return n
}

// Extract the first element from TlvBytes.
// Return the first element or nil if not found, and any remaining bytes.
func (tb TlvBytes) ExtractElement() (element TlvBytes, tail TlvBytes) {
	if _, tail = tb.DecodeVarNum(); tail == nil {
		return nil, tb
	}
	var length uint64
	if length, tail = tail.DecodeVarNum(); tail == nil || len(tail) < int(length) {
		return nil, tb
	}
	tail = tail[int(length):]
	element = tb[:len(tb)-len(tail)]
	return element, tail
}

// Split TlvBytes into elements.
// Return slice of elements, or nil if incomplete.
func (tb TlvBytes) SplitElements() (elements []TlvBytes) {
	elements = make([]TlvBytes, 0)
	for len(tb) > 0 {
		var element TlvBytes
		element, tb = tb.ExtractElement()
		if element == nil {
			return nil
		}
		elements = append(elements, element)
	}
	return elements
}

func JoinTlvBytes(s []TlvBytes) TlvBytes {
	return TlvBytes(bytes.Join(*(*[][]byte)(unsafe.Pointer(&s)), nil))
}

func EncodeVarNum(n uint64) TlvBytes {
	buf := make([]byte, int(C.SizeofVarNum(C.uint64_t(n))))
	C.EncodeVarNum((*C.uint8_t)(unsafe.Pointer(&buf[0])), C.uint64_t(n))
	return buf
}

func EncodeTlvTypeLength(tlvType TlvType, tlvLength int) TlvBytes {
	return JoinTlvBytes([]TlvBytes{
		EncodeVarNum(uint64(tlvType)),
		EncodeVarNum(uint64(tlvLength))})
}

func EncodeTlv(tlvType TlvType, value TlvBytes) TlvBytes {
	return JoinTlvBytes([]TlvBytes{
		EncodeVarNum(uint64(tlvType)),
		EncodeVarNum(uint64(len(value))),
		value})
}
