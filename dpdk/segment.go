package dpdk

/*
#include "mbuf.h"
*/
import "C"
import (
	"errors"
	"reflect"
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
	next := s.c.next
	return Segment{Mbuf{next}, s.pkt}, next != nil
}

func (s Segment) Len() int {
	return int(s.c.data_len)
}

// Get pointer to segment data.
func (s Segment) GetData() unsafe.Pointer {
	return unsafe.Pointer(uintptr(s.c.buf_addr) + uintptr(s.c.data_off))
}

// Map segment data as []byte.
func (s Segment) AsByteSlice() []byte {
	sh := reflect.SliceHeader{uintptr(s.GetData()), s.Len(), s.Len() + s.GetTailroom()}
	return *(*[]byte)(unsafe.Pointer(&sh))
}

func (s Segment) GetHeadroom() int {
	return int(C.rte_pktmbuf_headroom(s.c))
}

func (s Segment) SetHeadroom(headroom int) error {
	if s.Len() > 0 {
		return errors.New("cannot change headroom of non-empty segment")
	}
	if C.uint16_t(headroom) > s.c.buf_len {
		return errors.New("headroom cannot exceed buffer length")
	}
	s.c.data_off = C.uint16_t(headroom)
	return nil
}

func (s Segment) GetTailroom() int {
	return int(C.rte_pktmbuf_tailroom(s.c))
}

// Prepend in headroom.
// Return pointer to new space.
func (s Segment) Prepend(input []byte) error {
	count := len(input)
	if count == 0 {
		return nil
	}
	if count > s.GetHeadroom() {
		return errors.New("insufficient headroom")
	}
	s.c.data_off -= C.uint16_t(count)
	s.c.data_len += C.uint16_t(count)
	s.pkt.c.pkt_len += C.uint32_t(count)
	C.rte_memcpy(s.GetData(), unsafe.Pointer(&input[0]), C.size_t(count))
	return nil
}

// Remove len octets from head.
func (s Segment) Adj(len int) error {
	if len > s.Len() {
		return errors.New("segment shorter than adj amount")
	}
	s.c.data_off = s.c.data_off + C.uint16_t(len)
	s.c.data_len = s.c.data_len - C.uint16_t(len)
	s.pkt.c.pkt_len = s.pkt.c.pkt_len - C.uint32_t(len)
	return nil
}

// Append in tailroom.
func (s Segment) Append(input []byte) error {
	count := len(input)
	if count == 0 {
		return nil
	}
	if count > s.GetTailroom() {
		return errors.New("insufficient tailroom")
	}

	tail := unsafe.Pointer(uintptr(s.c.buf_addr) + uintptr(s.c.data_off) +
		uintptr(s.c.data_len))
	s.c.data_len += C.uint16_t(count)
	s.pkt.c.pkt_len += C.uint32_t(count)

	// skip memcpy if input is obtained from s.AsByteSlice()
	if tail != unsafe.Pointer(&input[0]) {
		C.rte_memcpy(tail, unsafe.Pointer(&input[0]), C.size_t(count))
	}
	return nil
}

// Remove count octets from tail.
func (s Segment) Trim(count int) error {
	if count > s.Len() {
		return errors.New("segment shorter than trim amount")
	}
	s.c.data_len = s.c.data_len - C.uint16_t(count)
	s.pkt.c.pkt_len = s.pkt.c.pkt_len - C.uint32_t(count)
	return nil
}
