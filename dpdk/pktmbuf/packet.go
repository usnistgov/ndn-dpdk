package pktmbuf

/*
#include "../../csrc/dpdk/mbuf.h"
#include "../../csrc/dpdk/mbuf-loc.h"
*/
import "C"
import (
	"errors"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// DefaultHeadroom is the default headroom of a mbuf.
const DefaultHeadroom = C.RTE_PKTMBUF_HEADROOM

// Packet represents a packet in mbuf.
type Packet C.struct_rte_mbuf

// PacketFromPtr converts *C.struct_rte_mbuf pointer to Packet.
func PacketFromPtr(ptr unsafe.Pointer) *Packet {
	return (*Packet)(ptr)
}

// Ptr returns *C.struct_rte_mbuf pointer.
func (pkt *Packet) Ptr() unsafe.Pointer {
	return unsafe.Pointer(pkt)
}

func (pkt *Packet) ptr() *C.struct_rte_mbuf {
	return (*C.struct_rte_mbuf)(pkt)
}

// Close releases the mbuf to mempool.
func (pkt *Packet) Close() error {
	C.rte_pktmbuf_free(pkt.ptr())
	return nil
}

// Len returns packet length in octets.
func (pkt *Packet) Len() int {
	return int(pkt.ptr().pkt_len)
}

// Port returns ingress network interface.
func (pkt *Packet) Port() uint16 {
	return uint16(pkt.ptr().port)
}

// SetPort sets ingress network interface.
func (pkt *Packet) SetPort(port uint16) {
	pkt.ptr().port = C.uint16_t(port)
}

// Timestamp returns receive timestamp.
func (pkt *Packet) Timestamp() eal.TscTime {
	return eal.TscTime(pkt.ptr().timestamp)
}

// SetTimestamp sets receive timestamp.
func (pkt *Packet) SetTimestamp(t eal.TscTime) {
	pkt.ptr().timestamp = C.uint64_t(t)
}

// IsSegmented returns true if the packet has more than one segment.
func (pkt *Packet) IsSegmented() bool {
	return pkt.ptr().nb_segs > 1
}

// DataPtr returns void* pointer to data in first segment.
func (pkt *Packet) DataPtr() unsafe.Pointer {
	pktC := pkt.ptr()
	return unsafe.Pointer(uintptr(pktC.buf_addr) + uintptr(pktC.data_off))
}

// ReadAll copies all octets into new []byte.
func (pkt *Packet) ReadAll() []byte {
	b := make([]byte, pkt.Len())
	pi := NewPacketIterator(pkt)
	pi.Read(b)
	return b
}

// Headroom returns headroom of the first segment.
func (pkt *Packet) Headroom() int {
	return int(C.rte_pktmbuf_headroom(pkt.ptr()))
}

// SetHeadroom changes headroom of the first segment.
// It can only be used on an empty packet.
func (pkt *Packet) SetHeadroom(headroom int) error {
	if pkt.Len() > 0 {
		return errors.New("cannot change headroom of non-empty packet")
	}
	pktC := pkt.ptr()
	if C.uint16_t(headroom) > pktC.buf_len {
		return errors.New("headroom cannot exceed buffer length")
	}
	pktC.data_off = C.uint16_t(headroom)
	return nil
}

// Tailroom returns tailroom of the last segment.
func (pkt *Packet) Tailroom() int {
	return int(C.rte_pktmbuf_tailroom(C.rte_pktmbuf_lastseg(pkt.ptr())))
}

// Prepend prepends to the packet in headroom of the first segment.
func (pkt *Packet) Prepend(input []byte) error {
	count := len(input)
	if count == 0 {
		return nil
	}

	room := C.rte_pktmbuf_prepend(pkt.ptr(), C.uint16_t(count))
	if room == nil {
		return errors.New("insufficient headroom")
	}
	C.rte_memcpy(unsafe.Pointer(room), unsafe.Pointer(&input[0]), C.size_t(count))
	return nil
}

// Append appends to the packet in tailroom of the last segment.
func (pkt *Packet) Append(input []byte) error {
	count := len(input)
	if count == 0 {
		return nil
	}

	room := C.rte_pktmbuf_append(pkt.ptr(), C.uint16_t(count))
	if room == nil {
		return errors.New("insufficient tailroom")
	}
	C.rte_memcpy(unsafe.Pointer(room), unsafe.Pointer(&input[0]), C.size_t(count))
	return nil
}

// Chain combines two packets.
// tail will be freed when pkt is freed.
func (pkt *Packet) Chain(tail *Packet) error {
	pktC := pkt.ptr()
	res := C.Packet_Chain(pktC, C.rte_pktmbuf_lastseg(pktC), tail.ptr())
	if res != 0 {
		return errors.New("too many segments")
	}
	return nil
}

// DeleteRange deletes len octets starting from offset.
// offset can be int, PacketIterator, or *PacketIterator.
func (pkt *Packet) DeleteRange(offset interface{}, len int) {
	pi := makePacketIteratorFromOffset(pkt, offset)
	C.MbufLoc_Delete(&pi.ml, C.uint32_t(len), pkt.ptr(), nil)
}

// LinearizeRange ensures two offsets are in the same mbuf.
// first and last can be int, PacketIterator, or *PacketIterator.
// Returns a C pointer to the octets in consecutive memory.
func (pkt *Packet) LinearizeRange(first interface{}, last interface{}, mp *Pool) error {
	firstPi := makePacketIteratorFromOffset(pkt, first)
	lastPi := makePacketIteratorFromOffset(pkt, last)
	n := firstPi.ml.rem - lastPi.ml.rem
	res := C.MbufLoc_Linearize(&firstPi.ml, &lastPi.ml, C.uint32_t(n), pkt.ptr(), mp.ptr())
	if res == nil {
		return eal.GetErrno()
	}
	return nil
}
