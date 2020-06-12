package pktmbuf

/*
#include "loc.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk/eal"
)

// PacketIterator represents an iterator over a packet.
type PacketIterator struct {
	ml C.MbufLoc
}

// NewPacketIterator creates a PacketIterator.
func NewPacketIterator(pkt *Packet) PacketIterator {
	var it PacketIterator
	C.MbufLoc_Init(&it.ml, pkt.getPtr())
	return it
}

// Reuse or create PacketIterator from an offset.
// offset: *PacketIterator or PacketIterator or int.
func makePacketIteratorFromOffset(pkt *Packet, offset interface{}) (pi *PacketIterator) {
	switch v := offset.(type) {
	case *PacketIterator:
		pi = v
	case PacketIterator:
		pi = &v
	case int:
		newPi := NewPacketIterator(pkt)
		pi = &newPi
		pi.Advance(v)
	default:
		panic("bad offset type")
	}
	return pi
}

// GetPtr returns *C.MbufLoc pointer.
func (it *PacketIterator) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(&it.ml)
}

// IsEnd returns true if the iterator is at the end of the input range.
func (it *PacketIterator) IsEnd() bool {
	return bool(C.MbufLoc_IsEnd(&it.ml))
}

// Advance advances the iterator by n octets.
func (it *PacketIterator) Advance(n int) int {
	return int(C.MbufLoc_Advance(&it.ml, C.uint32_t(n)))
}

// MakeIndirect clones next n octets into indirect mbufs.
func (it *PacketIterator) MakeIndirect(n int, mp *Pool) (*Packet, error) {
	res := C.MbufLoc_MakeIndirect(&it.ml, C.uint32_t(n), mp.getPtr())
	if res == nil {
		return nil, eal.GetErrno()
	}
	return PacketFromPtr(unsafe.Pointer(res)), nil
}

// Read copies next len(output) octets into output.
// Returns number of octets read.
func (it *PacketIterator) Read(output []byte) int {
	if len(output) == 0 {
		return 0
	}
	return int(C.MbufLoc_ReadTo(&it.ml, unsafe.Pointer(&output[0]), C.uint32_t(len(output))))
}

// PeekOctet returns next octet without advancing the iterator.
// Returns -1 if the iterator is at end of packet.
func (it *PacketIterator) PeekOctet() int {
	return int(C.MbufLoc_PeekOctet(&it.ml))
}
