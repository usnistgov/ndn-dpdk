package mbuftestenv

/*
#include "../../../csrc/dpdk/mbuf.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

// ListSegmentLengths returns a list of segment lengths in the packet.
func ListSegmentLengths(pkt *pktmbuf.Packet) (list []int) {
	for m := (*C.struct_rte_mbuf)(pkt.GetPtr()); m != nil; m = m.next {
		list = append(list, int(m.data_len))
	}
	return list
}
