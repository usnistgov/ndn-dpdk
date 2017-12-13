package dpdk

/*
#include "mbuf.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

type Mbuf struct {
	ptr *C.struct_rte_mbuf
	// DO NOT add other fields: *Mbuf is passed to C code as rte_mbuf**
}

func (m Mbuf) IsValid() bool {
	return m.ptr != nil
}

func (m Mbuf) Close() {
	C.rte_pktmbuf_free(m.ptr)
}

func (m Mbuf) AsPacket() Packet {
	return Packet{m}
}

type Packet struct {
	Mbuf
	// DO NOT add other fields: *Packet is passed to C code as rte_mbuf**
}

func (pkt Packet) Len() uint {
	return uint(pkt.ptr.pkt_len)
}

func (pkt Packet) CountSegments() uint {
	return uint(pkt.ptr.nb_segs)
}

func (pkt Packet) GetFirstSegment() Segment {
	return Segment{pkt.Mbuf, pkt}
}

func (pkt Packet) GetSegment(i uint) (Segment, error) {
	s := pkt.GetFirstSegment()
	for j := uint(0); j < i; j++ {
		ok := false
		s, ok = s.GetNext()
		if !ok {
			return s, errors.New("segment index out of range")
		}
	}
	return s, nil
}

func (pkt Packet) GetLastSegment() Segment {
	return Segment{Mbuf{C.rte_pktmbuf_lastseg(pkt.ptr)}, pkt}
}

// Append a segment.
// m: allocated Mbuf for new segment
// tail: if not nil, must be pkt.GetLastSegment(), to use faster implementation
// Return the new tail segment.
func (pkt Packet) AppendSegment(m Mbuf, tail *Segment) (Segment, error) {
	if tail == nil {
		res := C.rte_pktmbuf_chain(pkt.ptr, m.ptr)
		if res != 0 {
			return Segment{}, errors.New("too many segments")
		}
		return Segment{m, pkt}, nil
	}

	if pkt.CountSegments()+uint(m.ptr.nb_segs) > uint(C.RTE_MBUF_MAX_NB_SEGS) {
		return Segment{}, errors.New("too many segments")
	}

	tail.ptr.next = m.ptr
	pkt.ptr.nb_segs = pkt.ptr.nb_segs + m.ptr.nb_segs
	pkt.ptr.pkt_len = pkt.ptr.pkt_len + m.ptr.pkt_len
	m.ptr.nb_segs = 1
	m.ptr.pkt_len = 0
	return Segment{m, pkt}, nil
}

// Append all segments in pkt2.
// tail: if not nil, must be pkt.GetLastSegment(), to use faster implementation
func (pkt Packet) AppendPacket(pkt2 Packet, tail *Segment) error {
	_, e := pkt.AppendSegment(pkt2.Mbuf, tail)
	return e
}

// Read len octets at offset.
// buf: a buffer on C memory of at least len, for copying in case range is split between segments.
// Return a C pointer to the octets, either in segment or copied.
func (pkt Packet) Read(offset uint, len uint, buf unsafe.Pointer) (unsafe.Pointer, error) {
	res := C.rte_pktmbuf_read(pkt.ptr, C.uint32_t(offset), C.uint32_t(len), buf)
	if res == nil {
		return nil, errors.New("rte_pktmbuf_read out of range")
	}
	return res, nil
}

type PacketIterator struct {
	ml C.MbufLoc
}

func NewPacketIterator(pkt Packet) PacketIterator {
	var it PacketIterator
	C.MbufLoc_Init(&it.ml, pkt.ptr)
	return it
}

func NewPacketIteratorBounded(pkt Packet, off uint, len uint) PacketIterator {
	it := NewPacketIterator(pkt)
	it.Advance(off)
	it.ml.rem = C.uint32_t(len)
	return it
}

// Get native *C.MbufLoc pointer to use in other packages.
func (it *PacketIterator) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(&it.ml)
}

func (it *PacketIterator) IsEnd() bool {
	return bool(C.MbufLoc_IsEnd(&it.ml))
}

// Compute distance from it to it2.
// it.Advance(dist) equals it2 if dist is positive;
// it2.Advance(-dist) equals it if dist is negative.
func (it *PacketIterator) ComputeDistance(it2 *PacketIterator) int {
	return int(C.MbufLoc_Diff(&it.ml, &it2.ml))
}

func (it *PacketIterator) Advance(n uint) uint {
	return uint(C.MbufLoc_Advance(&it.ml, C.uint32_t(n)))
}

func (it *PacketIterator) Read(output []byte) uint {
	return uint(C.MbufLoc_Read(&it.ml, unsafe.Pointer(&output[0]), C.uint32_t(len(output))))
}

type Segment struct {
	Mbuf
	pkt Packet
}

func (s Segment) GetPacket() Packet {
	return s.pkt
}

// Get next segment.
// Return the next segment if it exists, and whether the next segment exists.
func (s Segment) GetNext() (Segment, bool) {
	next := s.ptr.next
	return Segment{Mbuf{next}, s.pkt}, next != nil
}

func (s Segment) Len() uint {
	return uint(s.ptr.data_len)
}

// Get pointer to segment data.
func (s Segment) GetData() unsafe.Pointer {
	return unsafe.Pointer(uintptr(s.ptr.buf_addr) + uintptr(s.ptr.data_off))
}

func (s Segment) GetHeadroom() uint {
	return uint(C.rte_pktmbuf_headroom(s.ptr))
}

func (s Segment) SetHeadroom(headroom uint) error {
	if s.Len() > 0 {
		return errors.New("cannot change headroom of non-empty segment")
	}
	if C.uint16_t(headroom) > s.ptr.buf_len {
		return errors.New("headroom cannot exceed buffer length")
	}
	s.ptr.data_off = C.uint16_t(headroom)
	return nil
}

func (s Segment) GetTailroom() uint {
	return uint(C.rte_pktmbuf_tailroom(s.ptr))
}

// Prepend len octets.
// Return pointer to new space.
func (s Segment) Prepend(len uint) (unsafe.Pointer, error) {
	if len > s.GetHeadroom() {
		return nil, errors.New("insufficient headroom")
	}
	s.ptr.data_off = s.ptr.data_off - C.uint16_t(len)
	s.ptr.data_len = s.ptr.data_len + C.uint16_t(len)
	s.pkt.ptr.pkt_len = s.pkt.ptr.pkt_len + C.uint32_t(len)
	return s.GetData(), nil
}

// Remove len octets from pkt.
// Return pointer to new pkt.
func (s Segment) Adj(len uint) (unsafe.Pointer, error) {
	if len > s.Len() {
		return nil, errors.New("segment shorter than adj amount")
	}
	s.ptr.data_off = s.ptr.data_off + C.uint16_t(len)
	s.ptr.data_len = s.ptr.data_len - C.uint16_t(len)
	s.pkt.ptr.pkt_len = s.pkt.ptr.pkt_len - C.uint32_t(len)
	return s.GetData(), nil
}

// Append len octets at tail.
// Return pointer to new space.
func (s Segment) Append(len uint) (unsafe.Pointer, error) {
	if len > s.GetTailroom() {
		return nil, errors.New("insufficient tailroom")
	}
	tail := unsafe.Pointer(uintptr(s.ptr.buf_addr) + uintptr(s.ptr.data_off) +
		uintptr(s.ptr.data_len))
	s.ptr.data_len = s.ptr.data_len + C.uint16_t(len)
	s.pkt.ptr.pkt_len = s.pkt.ptr.pkt_len + C.uint32_t(len)
	return tail, nil
}

// Remove len octets from tail.
func (s Segment) Trim(len uint) error {
	if len > s.Len() {
		return errors.New("segment shorter than trim amount")
	}
	s.ptr.data_len = s.ptr.data_len - C.uint16_t(len)
	s.pkt.ptr.pkt_len = s.pkt.ptr.pkt_len - C.uint32_t(len)
	return nil
}
