package ndn

/*
#include "tlv-decode-pos.h"
*/
import "C"
import (
	"encoding/binary"

	"ndn-dpdk/dpdk"
)

type TlvDecodePos struct {
	it dpdk.PacketIterator
}

func NewTlvDecodePos(pkt dpdk.IMbuf) TlvDecodePos {
	return TlvDecodePos{dpdk.NewPacketIterator(pkt)}
}

func (d *TlvDecodePos) getPtr() *C.TlvDecodePos {
	return (*C.TlvDecodePos)(d.it.GetPtr())
}

// Decode a TLV-TYPE or TLV-LENGTH number.
func (d *TlvDecodePos) ReadVarNum() (v uint64, e error) {
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
