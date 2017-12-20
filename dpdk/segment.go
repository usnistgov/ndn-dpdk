package dpdk

/*
#include "mbuf.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

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

func (s Segment) Len() int {
	return int(s.ptr.data_len)
}

// Get pointer to segment data.
func (s Segment) GetData() unsafe.Pointer {
	return unsafe.Pointer(uintptr(s.ptr.buf_addr) + uintptr(s.ptr.data_off))
}

func (s Segment) GetHeadroom() int {
	return int(C.rte_pktmbuf_headroom(s.ptr))
}

func (s Segment) SetHeadroom(headroom int) error {
	if s.Len() > 0 {
		return errors.New("cannot change headroom of non-empty segment")
	}
	if C.uint16_t(headroom) > s.ptr.buf_len {
		return errors.New("headroom cannot exceed buffer length")
	}
	s.ptr.data_off = C.uint16_t(headroom)
	return nil
}

func (s Segment) GetTailroom() int {
	return int(C.rte_pktmbuf_tailroom(s.ptr))
}

// Prepend len octets.
// Return pointer to new space.
func (s Segment) Prepend(len int) (unsafe.Pointer, error) {
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
func (s Segment) Adj(len int) (unsafe.Pointer, error) {
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
func (s Segment) Append(len int) (unsafe.Pointer, error) {
	if len > s.GetTailroom() {
		return nil, errors.New("insufficient tailroom")
	}
	tail := unsafe.Pointer(uintptr(s.ptr.buf_addr) + uintptr(s.ptr.data_off) +
		uintptr(s.ptr.data_len))
	s.ptr.data_len = s.ptr.data_len + C.uint16_t(len)
	s.pkt.ptr.pkt_len = s.pkt.ptr.pkt_len + C.uint32_t(len)
	return tail, nil
}

// Append octets at tail.
func (s Segment) AppendOctets(input []byte) error {
	buf, e := s.Append(len(input))
	if e != nil {
		return e
	}

	for i, b := range input {
		ptr := unsafe.Pointer(uintptr(buf) + uintptr(i))
		*(*byte)(ptr) = b
	}
	return nil
}

// Remove len octets from tail.
func (s Segment) Trim(len int) error {
	if len > s.Len() {
		return errors.New("segment shorter than trim amount")
	}
	s.ptr.data_len = s.ptr.data_len - C.uint16_t(len)
	s.pkt.ptr.pkt_len = s.pkt.ptr.pkt_len - C.uint32_t(len)
	return nil
}
