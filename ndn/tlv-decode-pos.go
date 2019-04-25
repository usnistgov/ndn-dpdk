package ndn

/*
#include "tlv-decode-pos.h"
*/
import "C"
import (
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
func (d *TlvDecodePos) ReadVarNum() (v uint32, e error) {
	if res := C.DecodeVarNum(d.getPtr(), (*C.uint32_t)(&v)); res != C.NdnError_OK {
		return 0, NdnError(res)
	}
	return v, nil
}
