package dpdk

/*
#include "mbuf-loc.h"
*/
import "C"
import (
	"unsafe"
)

type PacketIterator struct {
	ml C.MbufLoc
}

func NewPacketIterator(pkt IMbuf) PacketIterator {
	var it PacketIterator
	C.MbufLoc_Init(&it.ml, (*C.struct_rte_mbuf)(pkt.GetPtr()))
	return it
}

func NewPacketIteratorBounded(pkt Packet, off int, len int) PacketIterator {
	it := NewPacketIterator(pkt)
	it.Advance(off)
	it.ml.rem = C.uint32_t(len)
	return it
}

// Reuse or create PacketIterator from an offset.
// offset: *PacketIterator or PacketIterator or int.
func makePacketIteratorFromOffset(pkt Packet, offset interface{}) (pi *PacketIterator) {
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

// Get native *C.MbufLoc pointer to use in other packages.
func (it *PacketIterator) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(&it.ml)
}

func (it *PacketIterator) IsEnd() bool {
	return bool(C.MbufLoc_IsEnd(&it.ml))
}

func (it *PacketIterator) Advance(n int) int {
	return int(C.MbufLoc_Advance(&it.ml, C.uint32_t(n)))
}

// Compute distance from it to it2.
// it.Advance(dist) equals it2 if dist is positive;
// it2.Advance(-dist) equals it if dist is negative.
func (it *PacketIterator) ComputeDistance(it2 PacketIterator) int {
	return int(C.MbufLoc_Diff(&it.ml, &it2.ml))
}

// Clone next n octets into indirect mbufs.
func (it *PacketIterator) MakeIndirect(n int, mp PktmbufPool) (Packet, error) {
	res := C.MbufLoc_MakeIndirect(&it.ml, C.uint32_t(n), mp.c)
	if res == nil {
		return Packet{}, GetErrno()
	}
	return Mbuf{res}.AsPacket(), nil
}

func (it *PacketIterator) Read(output []byte) int {
	if len(output) == 0 {
		return 0
	}
	return int(C.MbufLoc_ReadTo(&it.ml, unsafe.Pointer(&output[0]), C.uint32_t(len(output))))
}

// Peek next octet.
// Return -1 if end of packet.
func (it *PacketIterator) PeekOctet() int {
	return int(C.MbufLoc_PeekOctet(&it.ml))
}
