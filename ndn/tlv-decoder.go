package ndn

/*
#include "tlv-decoder.h"
*/
import "C"
import (
	"encoding/binary"

	"ndn-dpdk/dpdk"
)

type TlvDecoder struct {
	it dpdk.PacketIterator
}

func NewTlvDecoder(pkt dpdk.Packet) TlvDecoder {
	return TlvDecoder{dpdk.NewPacketIterator(pkt)}
}

func (d *TlvDecoder) getPtr() *C.TlvDecoder {
	return (*C.TlvDecoder)(d.it.GetPtr())
}

// Decode a TLV-TYPE or TLV-LENGTH number.
func (d *TlvDecoder) ReadVarNum() (v uint64, e error) {
	res := C.DecodeVarNum(d.getPtr(), (*C.uint64_t)(&v))
	if res != C.NdnError_OK {
		return 0, NdnError(res)
	}
	return v, nil
}

// Decode a TLV-TYPE or TLV-LENGTH number from Go buffer.
func DecodeVarNum(input []byte) (v uint64, size int, ok bool) {
	if len(input) < 1 {
		return 0, 0, false
	}
	switch input[0] {
	case 253:
		if len(input) < 3 {
			return 0, 0, false
		}
		return uint64(binary.BigEndian.Uint16(input[1:3])), 3, true
	case 254:
		if len(input) < 5 {
			return 0, 0, false
		}
		return uint64(binary.BigEndian.Uint32(input[1:5])), 5, true
	case 255:
		if len(input) < 9 {
			return 0, 0, false
		}
		return binary.BigEndian.Uint64(input[1:9]), 9, true
	}
	return uint64(input[0]), 1, true
}
