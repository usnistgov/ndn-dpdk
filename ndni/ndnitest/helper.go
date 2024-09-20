package ndnitest

/*
#include "../../csrc/ndni/packet.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

var (
	makeAR       = testenv.MakeAR
	bytesFromHex = testenv.BytesFromHex
	bytesEqual   = testenv.BytesEqual
	nameEqual    = ndntestenv.NameEqual

	directDataroom int
)

type packet struct {
	*pktmbuf.Packet
	N     *ndni.Packet
	mbuf  *C.struct_rte_mbuf
	mbufA *pktmbuf.MbufAccessor
	npkt  *C.Packet
}

func makePacket(args ...any) (p *packet) {
	p = toPacket(mbuftestenv.MakePacket(args...).Ptr())
	*C.Packet_GetLpL3Hdr(p.npkt) = C.LpL3{}
	return p
}

func toPacket(ptr unsafe.Pointer) (p *packet) {
	if ptr == nil {
		return nil
	}
	p = &packet{
		N:    ndni.PacketFromPtr(ptr),
		mbuf: (*C.struct_rte_mbuf)(ptr),
	}
	p.mbufA = pktmbuf.MbufAccessorFromPtr(p.mbuf)
	p.Packet = p.N.Mbuf()
	p.npkt = C.Packet_FromMbuf(p.mbuf)
	return p
}
