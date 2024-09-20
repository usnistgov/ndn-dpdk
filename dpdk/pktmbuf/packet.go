// Package pktmbuf contains bindings of DPDK mbuf library.
package pktmbuf

/*
#include "../../csrc/dpdk/mbuf.h"
enum {
	c_offsetof_Mbuf_DataOff = offsetof(struct rte_mbuf, data_off),
	c_offsetof_Mbuf_NbSegs = offsetof(struct rte_mbuf, nb_segs),
	c_offsetof_Mbuf_Port = offsetof(struct rte_mbuf, port),
	c_offsetof_Mbuf_PacketType = offsetof(struct rte_mbuf, packet_type),
	c_offsetof_Mbuf_PktLen = offsetof(struct rte_mbuf, pkt_len),
	c_offsetof_Mbuf_DataLen = offsetof(struct rte_mbuf, data_len),
	c_offsetof_Mbuf_BufLen = offsetof(struct rte_mbuf, buf_len),
};
*/
import "C"
import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

var logger = logging.New("pktmbuf")

// DefaultHeadroom is the default headroom of a mbuf.
const DefaultHeadroom = C.RTE_PKTMBUF_HEADROOM

// Packet represents a packet in mbuf.
type Packet C.struct_rte_mbuf

func (pkt *Packet) mbuf() *MbufAccessor {
	return MbufAccessorFromPtr(pkt)
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
	return int(*pkt.mbuf().PktLen())
}

// Port returns ingress network interface.
func (pkt *Packet) Port() uint16 {
	return uint16(*pkt.mbuf().Port())
}

// SetPort sets ingress network interface.
func (pkt *Packet) SetPort(port uint16) {
	*pkt.mbuf().Port() = uint16(port)
}

// Timestamp returns receive timestamp.
func (pkt *Packet) Timestamp() eal.TscTime {
	return eal.TscTime(C.Mbuf_GetTimestamp(pkt.ptr()))
}

// SetTimestamp sets receive timestamp.
func (pkt *Packet) SetTimestamp(t eal.TscTime) {
	C.Mbuf_SetTimestamp(pkt.ptr(), C.TscTime(t))
}

// Type32 returns 32-bit packet type.
func (pkt *Packet) Type32() uint32 {
	return *pkt.mbuf().PacketType()
}

// SetType32 sets 32-bit packet type.
func (pkt *Packet) SetType32(packetType uint32) {
	*pkt.mbuf().PacketType() = packetType
}

// SegmentBytes returns the data in each segment.
// Each []byte aliases the mbuf.
func (pkt *Packet) SegmentBytes() (list [][]byte) {
	list = make([][]byte, 0, *pkt.mbuf().NbSegs())
	for m := (*MbufAccessor)(pkt.ptr()); m != nil; m = (*MbufAccessor)(m.next) {
		buf := unsafe.Slice((*byte)(m.buf_addr), *m.BufLen())
		list = append(list, buf[*m.DataOff():*m.DataOff()+*m.DataLen()])
	}
	return list
}

// Bytes returns a []byte that contains a copy of the data in this packet.
func (pkt *Packet) Bytes() []byte {
	return bytes.Join(pkt.SegmentBytes(), nil)
}

// Headroom returns headroom of the first segment.
func (pkt *Packet) Headroom() int {
	return int(*pkt.mbuf().DataOff())
}

// SetHeadroom changes headroom of the first segment.
// It can only be used on an empty packet.
func (pkt *Packet) SetHeadroom(headroom int) error {
	if pkt.Len() > 0 {
		return errors.New("cannot change headroom of non-empty packet")
	}
	if uint16(headroom) > *pkt.mbuf().BufLen() {
		return errors.New("headroom cannot exceed buffer length")
	}
	*pkt.mbuf().DataOff() = uint16(headroom)
	return nil
}

// Tailroom returns tailroom of the last segment.
func (pkt *Packet) Tailroom() int {
	if *pkt.mbuf().NbSegs() == 1 {
		return int(*pkt.mbuf().BufLen() - *pkt.mbuf().DataOff() - *pkt.mbuf().DataLen())
	}
	return int(C.rte_pktmbuf_tailroom(C.rte_pktmbuf_lastseg(pkt.ptr())))
}

// ReadFrom reads once from the reader into the dataroom of this packet.
// It can only be used on an empty packet.
func (pkt *Packet) ReadFrom(r io.Reader) (n int64, e error) {
	bufs := pkt.SegmentBytes()
	if len(bufs) > 1 || len(bufs[0]) > 0 {
		return 0, errors.New("cannot ReadFrom on non-empty packet")
	}
	room := bufs[0][:cap(bufs[0])]
	ni, e := r.Read(room)
	if ni > 0 {
		*pkt.mbuf().PktLen() = uint32(ni)
		*pkt.mbuf().DataLen() = uint16(ni)
	}
	return int64(ni), e
}

// Prepend prepends to the packet in headroom of the first segment.
func (pkt *Packet) Prepend(input []byte) error {
	count := len(input)
	if count == 0 {
		return nil
	}

	room := C.rte_pktmbuf_prepend(pkt.ptr(), C.uint16_t(count))
	if room == nil {
		return fmt.Errorf("insufficient headroom %d", pkt.Headroom())
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(room)), count), input)
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
		return fmt.Errorf("insufficient tailroom %d", pkt.Tailroom())
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(room)), count), input)
	return nil
}

// Chain combines two packets.
// tail will be freed when pkt is freed.
func (pkt *Packet) Chain(tail *Packet) error {
	pktC := pkt.ptr()
	if !C.Mbuf_Chain(pktC, C.rte_pktmbuf_lastseg(pktC), tail.ptr()) {
		return errors.New("too many segments")
	}
	return nil
}

// PacketFromPtr converts *C.struct_rte_mbuf pointer to Packet.
func PacketFromPtr(ptr unsafe.Pointer) *Packet {
	return (*Packet)(ptr)
}

// MbufAccessor allows accessing mbuf union fields as pointers.
type MbufAccessor C.struct_rte_mbuf

func (m *MbufAccessor) DataOff() *uint16 {
	return (*uint16)(unsafe.Add(unsafe.Pointer(m), C.c_offsetof_Mbuf_DataOff))
}

func (m *MbufAccessor) NbSegs() *uint16 {
	return (*uint16)(unsafe.Add(unsafe.Pointer(m), C.c_offsetof_Mbuf_NbSegs))
}

func (m *MbufAccessor) Port() *uint16 {
	return (*uint16)(unsafe.Add(unsafe.Pointer(m), C.c_offsetof_Mbuf_Port))
}

func (m *MbufAccessor) PacketType() *uint32 {
	return (*uint32)(unsafe.Add(unsafe.Pointer(m), C.c_offsetof_Mbuf_PacketType))
}

func (m *MbufAccessor) PktLen() *uint32 {
	return (*uint32)(unsafe.Add(unsafe.Pointer(m), C.c_offsetof_Mbuf_PktLen))
}

func (m *MbufAccessor) DataLen() *uint16 {
	return (*uint16)(unsafe.Add(unsafe.Pointer(m), C.c_offsetof_Mbuf_DataLen))
}

func (m *MbufAccessor) BufLen() *uint16 {
	return (*uint16)(unsafe.Add(unsafe.Pointer(m), C.c_offsetof_Mbuf_BufLen))
}

// MbufAccessorFromPtr converts *C.struct_rte_mbuf pointer to Packet.
func MbufAccessorFromPtr[T any](ptr *T) *MbufAccessor {
	return (*MbufAccessor)(unsafe.Pointer(ptr))
}
