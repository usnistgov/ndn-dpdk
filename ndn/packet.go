package ndn

/*
#include "packet.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
)

// Get size of PacketPriv structure.
// PktmbufPool's privSize must be no less than this size.
func SizeofPacketPriv() uint16 {
	return uint16(C.sizeof_PacketPriv)
}

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
	return fmt.Sprintf("%d", int(t))
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
	return fmt.Sprintf("%d", int(t))
}

// NDN network layer packet with parsed LP and Interest/Data headers.
type Packet struct {
	c *C.Packet
}

func PacketFromDpdk(m dpdk.Packet) (pkt Packet) {
	if m.IsValid() {
		pkt.c = C.Packet_FromMbuf((*C.struct_rte_mbuf)(m.GetPtr()))
	}
	return pkt
}

// Construct Packet from *C.struct_rte_mbuf pointing to first segment.
// This function can accept nil pointer.
func PacketFromPtr(ptr unsafe.Pointer) (pkt Packet) {
	if ptr != nil {
		pkt.c = C.Packet_FromMbuf((*C.struct_rte_mbuf)(ptr))
	}
	return pkt
}

func (pkt Packet) AsDpdkPacket() dpdk.Packet {
	return dpdk.MbufFromPtr(unsafe.Pointer(pkt.c)).AsPacket()
}

func (pkt Packet) GetL2Type() L2PktType {
	return L2PktType(C.Packet_GetL2PktType(pkt.c))
}

func (pkt Packet) GetLpHdr() *LpPkt {
	return (*LpPkt)(unsafe.Pointer(C.Packet_GetLpHdr(pkt.c)))
}

func (pkt Packet) SetLpHdr(lpp LpPkt) {
	C.Packet_SetL2PktType(pkt.c, C.L2PktType_NdnlpV2)
	lpp1 := pkt.GetLpHdr()
	*lpp1 = lpp
}

func (pkt Packet) GetL3Type() L3PktType {
	return L3PktType(C.Packet_GetL3PktType(pkt.c))
}

func (pkt Packet) ParseL3() error {
	res := NdnError(C.Packet_ParseL3(pkt.c))
	if res != NdnError_OK {
		return res
	}
	return nil
}

func (pkt Packet) AsInterest() *InterestPkt {
	return (*InterestPkt)(unsafe.Pointer(C.Packet_GetInterestHdr(pkt.c)))
}

func (pkt Packet) AsData() *Data {
	return &Data{pkt, C.Packet_GetDataHdr(pkt.c)}
}
