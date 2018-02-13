package ndn

/*
#include "tlv-decode-pos.h"
*/
import "C"
import "ndn-dpdk/dpdk"

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
