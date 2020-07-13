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
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

var (
	makeAR       = testenv.MakeAR
	bytesFromHex = testenv.BytesFromHex
	bytesEqual   = testenv.BytesEqual
	nameEqual    = ndntestenv.NameEqual

	directDataroom int
	headerMp       *C.struct_rte_mempool
	indirectMp     *C.struct_rte_mempool
)

func initMempools() {
	directDataroom = ndni.PacketMempool.Config().Dataroom
	headerMp = (*C.struct_rte_mempool)(ndnitestenv.Header.Pool().Ptr())
	indirectMp = (*C.struct_rte_mempool)(mbuftestenv.Indirect.Pool().Ptr())
}

type packet struct {
	*pktmbuf.Packet
	mbuf *C.struct_rte_mbuf
	npkt *C.Packet
}

func makePacket(args ...interface{}) (p packet) {
	p.Packet = mbuftestenv.MakePacket(args...)
	p.mbuf = (*C.struct_rte_mbuf)(p.Ptr())
	p.npkt = C.Packet_FromMbuf(p.mbuf)
	*C.Packet_GetLpL3Hdr(p.npkt) = C.LpL3{}
	return p
}

func toPacket(ptr unsafe.Pointer) (p packet) {
	p.Packet = pktmbuf.PacketFromPtr(ptr)
	p.mbuf = (*C.struct_rte_mbuf)(ptr)
	p.npkt = C.Packet_FromMbuf(p.mbuf)
	return p
}
