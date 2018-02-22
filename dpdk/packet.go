package dpdk

/*
#include "mbuf.h"
#include "mbuf-loc.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

type Packet struct {
	Mbuf
	// DO NOT add other fields: *Packet is passed to C code as rte_mbuf**
}

func (pkt Packet) Len() int {
	return int(pkt.ptr.pkt_len)
}

func (pkt Packet) GetTimestamp() TscTime {
	return TscTime(pkt.ptr.timestamp)
}

func (pkt Packet) CountSegments() int {
	return int(pkt.ptr.nb_segs)
}

func (pkt Packet) GetFirstSegment() Segment {
	return Segment{pkt.Mbuf, pkt}
}

func (pkt Packet) GetSegment(i int) *Segment {
	s := pkt.GetFirstSegment()
	for j := 0; j < i; j++ {
		ok := false
		s, ok = s.GetNext()
		if !ok {
			return nil
		}
	}
	return &s
}

func (pkt Packet) GetLastSegment() Segment {
	return Segment{Mbuf{C.rte_pktmbuf_lastseg(pkt.ptr)}, pkt}
}

// Append a segment.
// m: allocated Mbuf for new segment
// tail: if not nil, must be pkt.GetLastSegment(), to use faster implementation
// Return the new tail segment.
func (pkt Packet) AppendSegmentHint(m Mbuf, tail *Segment) (Segment, error) {
	var res C.int
	if tail == nil {
		res = C.rte_pktmbuf_chain(pkt.ptr, m.ptr)
	} else {
		res = C.Packet_Chain(pkt.ptr, tail.ptr, m.ptr)
	}

	if res != 0 {
		return Segment{}, errors.New("too many segments")
	}
	return Segment{m, pkt}, nil
}

func (pkt Packet) AppendSegment(m Mbuf) (Segment, error) {
	return pkt.AppendSegmentHint(m, nil)
}

// Append all segments in pkt2.
// tail: if not nil, must be pkt.GetLastSegment(), to use faster implementation
func (pkt Packet) AppendPacketHint(pkt2 Packet, tail *Segment) error {
	_, e := pkt.AppendSegmentHint(pkt2.Mbuf, tail)
	return e
}

func (pkt Packet) AppendPacket(pkt2 Packet) error {
	return pkt.AppendPacketHint(pkt2, nil)
}

// Copy len(output) octets at offset into buf.
// Return actual number of octets read.
func (pkt Packet) ReadTo(offset int, output []byte) int {
	pi := NewPacketIterator(pkt)
	pi.Advance(offset)
	return pi.Read(output)
}

// Copy all octets into new []byte.
func (pkt Packet) ReadAll() []byte {
	b := make([]byte, pkt.Len())
	pkt.ReadTo(0, b)
	return b
}

// Delete len octets starting from offset (int or PacketIterator or *PacketIterator).
func (pkt Packet) DeleteRange(offset interface{}, len int) {
	pi := makePacketIteratorFromOffset(pkt, offset)
	C.MbufLoc_Delete(&pi.ml, C.uint32_t(len), pkt.ptr, nil)
}

// Ensure two offsets are in the same Mbuf.
// Return a C pointer to the octets in consecutive memory.
func (pkt Packet) LinearizeRange(first interface{}, last interface{}, mp PktmbufPool) (unsafe.Pointer, error) {
	firstPi := makePacketIteratorFromOffset(pkt, first)
	lastPi := makePacketIteratorFromOffset(pkt, last)
	res := C.MbufLoc_Linearize(&firstPi.ml, &lastPi.ml, pkt.ptr, mp.ptr)
	if res == nil {
		return nil, GetErrno()
	}
	return unsafe.Pointer(res), nil
}

func init() {
	var pkt Packet
	if unsafe.Sizeof(pkt) != unsafe.Sizeof(pkt.ptr) {
		panic("sizeof dpdk.Packet differs from *C.struct_rte_mbuf")
	}
}
