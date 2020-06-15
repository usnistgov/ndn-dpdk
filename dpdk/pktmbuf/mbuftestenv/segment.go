package mbuftestenv

/*
#include "../../../csrc/dpdk/mbuf.h"
*/
import "C"
import (
	"fmt"
	"ndn-dpdk/dpdk/pktmbuf"
	"unsafe"
)

// ListSegmentLengths returns a list of segment lengths in the packet.
func ListSegmentLengths(pkt *pktmbuf.Packet) (list []int) {
	for m := (*C.struct_rte_mbuf)(pkt.GetPtr()); m != nil; m = m.next {
		list = append(list, int(m.data_len))
	}
	return list
}

// PacketSplitTailSegment moves last n octets of last segment into a separate segment.
func PacketSplitTailSegment(pkt *pktmbuf.Packet, n int) *pktmbuf.Packet {
	segC := C.rte_pktmbuf_lastseg((*C.struct_rte_mbuf)(pkt.GetPtr()))
	if C.rte_pktmbuf_trim(segC, C.uint16_t(n)) != 0 {
		panic(fmt.Errorf("last segment has %d octets, cannot remove %d octets", segC.data_len, n))
	}
	data := C.rte_pktmbuf_mtod_offset_(segC, C.uint16_t(segC.data_len))

	tail := MakePacket()
	if tailroom := tail.GetTailroom(); tailroom < n {
		panic(fmt.Errorf("cannot append %d octets in tailroom %d", n, tailroom))
	}
	room := C.rte_pktmbuf_append((*C.struct_rte_mbuf)(tail.GetPtr()), C.uint16_t(n))
	C.rte_memcpy(unsafe.Pointer(room), data, C.size_t(n))

	pkt.Chain(tail)
	return pkt
}
