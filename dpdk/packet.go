package dpdk

/*
#include "mbuf.h"
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

func (pkt Packet) CountSegments() int {
	return int(pkt.ptr.nb_segs)
}

func (pkt Packet) GetFirstSegment() Segment {
	return Segment{pkt.Mbuf, pkt}
}

func (pkt Packet) GetSegment(i int) (Segment, error) {
	s := pkt.GetFirstSegment()
	for j := 0; j < i; j++ {
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

	if C.uint16_t(pkt.CountSegments())+m.ptr.nb_segs > C.RTE_MBUF_MAX_NB_SEGS {
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
func (pkt Packet) Read(offset int, len int, buf unsafe.Pointer) (unsafe.Pointer, error) {
	res := C.rte_pktmbuf_read(pkt.ptr, C.uint32_t(offset), C.uint32_t(len), buf)
	if res == nil {
		return nil, errors.New("rte_pktmbuf_read out of range")
	}
	return res, nil
}
