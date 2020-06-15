package ndn

/*
#include "../csrc/ndn/packet.h"
*/
import "C"
import (
	"fmt"
	"strconv"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

type L2PktType int

const (
	L2PktType_None    L2PktType = C.L2PktType_None
	L2PktType_NdnlpV2 L2PktType = C.L2PktType_NdnlpV2
)

func (t L2PktType) String() string {
	switch t {
	case L2PktType_NdnlpV2:
		return "NDNLPv2"
	}
	return strconv.Itoa(int(t))
}

type L3PktType int

const (
	L3PktType_None     L3PktType = C.L3PktType_None
	L3PktType_Interest L3PktType = C.L3PktType_Interest
	L3PktType_Data     L3PktType = C.L3PktType_Data
	L3PktType_Nack     L3PktType = C.L3PktType_Nack
)

func (t L3PktType) String() string {
	switch t {
	case L3PktType_Interest:
		return "Interest"
	case L3PktType_Data:
		return "Data"
	case L3PktType_Nack:
		return "Nack"
	}
	return strconv.Itoa(int(t))
}

// Packet represents a NDN network layer packet with parsed LP and Interest/Data headers.
type Packet C.Packet

// PacketFromPtr converts *C.Packet or *C.struct_rte_mbuf pointer to Packet.
func PacketFromPtr(ptr unsafe.Pointer) (pkt *Packet) {
	if ptr == nil {
		return nil
	}
	return (*Packet)(C.Packet_FromMbuf((*C.struct_rte_mbuf)(ptr)))
}

// PacketFromMbuf converts pktmbuf.Packet to Packet.
func PacketFromMbuf(m *pktmbuf.Packet) (pkt *Packet) {
	return PacketFromPtr(m.GetPtr())
}

// GetPtr returns *C.Packet or *C.struct_rte_mbuf pointer.
func (pkt *Packet) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(pkt)
}

func (pkt *Packet) getPtr() *C.Packet {
	return (*C.Packet)(pkt)
}

// AsMbuf converts to pktmbuf.Packet.
func (pkt *Packet) AsMbuf() *pktmbuf.Packet {
	return pktmbuf.PacketFromPtr(pkt.GetPtr())
}

// GetL2Type returns layer 2 packet type.
func (pkt *Packet) GetL2Type() L2PktType {
	return L2PktType(C.Packet_GetL2PktType(pkt.getPtr()))
}

// GetL3Type returns layer 3 packet type.
func (pkt *Packet) GetL3Type() L3PktType {
	return L3PktType(C.Packet_GetL3PktType(pkt.getPtr()))
}

// GetLpHdr returns NDNLP header.
// L2 must be parsed as NDNLP and L3 must be unparsed.
func (pkt *Packet) GetLpHdr() *LpHeader {
	return (*LpHeader)(unsafe.Pointer(C.Packet_GetLpHdr(pkt.getPtr())))
}

// GetLpL3 returns NDNLP layer 3 header.
// Packet must be parsed as NDNLP.
func (pkt *Packet) GetLpL3() *LpL3 {
	return (*LpL3)(unsafe.Pointer(C.Packet_GetLpL3Hdr(pkt.getPtr())))
}

// AsInterest converts to Interest type.
// Packet must be parsed as Interest.
func (pkt *Packet) AsInterest() *Interest {
	return &Interest{pkt, C.Packet_GetInterestHdr(pkt.getPtr())}
}

// AsData converts to Data type.
// Packet must be parsed as Data.
func (pkt *Packet) AsData() *Data {
	return &Data{pkt, C.Packet_GetDataHdr(pkt.getPtr())}
}

// AsNack converts to Nack type.
// Packet must be parsed as Nack.
func (pkt *Packet) AsNack() *Nack {
	return &Nack{pkt, C.Packet_GetNackHdr(pkt.getPtr())}
}

func (pkt *Packet) String() string {
	switch pkt.GetL3Type() {
	case L3PktType_Interest:
		return fmt.Sprintf("I %s", pkt.AsInterest())
	case L3PktType_Data:
		return fmt.Sprintf("D %s", pkt.AsData())
	case L3PktType_Nack:
		return fmt.Sprintf("N %s", pkt.AsNack())
	}
	return fmt.Sprintf("Packet(l3=%d)", pkt.GetL3Type())
}

// ParseL2 performs layer 2 parsing.
func (pkt *Packet) ParseL2() error {
	res := NdnError(C.Packet_ParseL2(pkt.getPtr()))
	if res != NdnError_OK {
		return res
	}
	return nil
}

// ParseL3 performs layer 3 parsing.
func (pkt *Packet) ParseL3(nameMp *pktmbuf.Pool) error {
	var mpC *C.struct_rte_mempool
	if nameMp != nil {
		mpC = (*C.struct_rte_mempool)(nameMp.GetPtr())
	}
	res := NdnError(C.Packet_ParseL3(pkt.getPtr(), mpC))
	if res != NdnError_OK {
		return res
	}
	return nil
}

// IL3Packet represents a layer 3 packet that allows conversion to Packet.
type IL3Packet interface {
	GetPacket() *Packet
}

// GetPacket implements IL3Packet interface.
func (pkt *Packet) GetPacket() *Packet {
	return pkt
}
