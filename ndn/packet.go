package ndn

/*
#include "packet.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// Get size of PacketPriv structure.
// PktmbufPool's privSize must be no less than this size.
func SizeofPacketPriv() uint16 {
	return uint16(C.sizeof_PacketPriv)
}

type L2PktType int

const (
	L2PktType_None    L2PktType = C.L2PktType_None
	L2PktType_NdnlpV2           = C.L2PktType_NdnlpV2
)

func (t L2PktType) String() string {
	switch t {
	case L2PktType_NdnlpV2:
		return "NDNLPv2"
	}
	return fmt.Sprintf("%d", int(t))
}

type NdnPktType int

const (
	NdnPktType_None     NdnPktType = C.NdnPktType_None
	NdnPktType_Interest            = C.NdnPktType_Interest
	NdnPktType_Data                = C.NdnPktType_Data
	NdnPktType_Nack                = C.NdnPktType_Nack
)

func (t NdnPktType) String() string {
	switch t {
	case NdnPktType_Interest:
		return "Interest"
	case NdnPktType_Data:
		return "Data"
	case NdnPktType_Nack:
		return "Nack"
	}
	return fmt.Sprintf("%d", int(t))
}

// NDN network layer packet with parsed LP and Interest/Data headers.
type Packet struct {
	c *C.Packet
}

// Construct Packet from *C.struct_rte_mbuf pointing to first segment.
// This function can accept nil pointer.
func PacketFromPtr(ptr unsafe.Pointer) (pkt Packet) {
	if ptr != nil {
		pkt.c = C.Packet_FromMbuf((*C.struct_rte_mbuf)(ptr))
	}
	return pkt
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

func (pkt Packet) GetNetType() NdnPktType {
	return NdnPktType(C.Packet_GetNdnPktType(pkt.c))
}

func (pkt Packet) AsInterest() *InterestPkt {
	return (*InterestPkt)(unsafe.Pointer(C.Packet_GetInterestHdr(pkt.c)))
}

func (pkt Packet) AsData() *DataPkt {
	return (*DataPkt)(unsafe.Pointer(C.Packet_GetDataHdr(pkt.c)))
}

func (pkt Packet) SetNetHdr(netp interface{}) {
	if interest, ok := netp.(*InterestPkt); ok {
		if pkt.GetL2Type() == L2PktType_NdnlpV2 && pkt.GetLpHdr().GetNackReason() != NackReason_None {
			C.Packet_SetNdnPktType(pkt.c, C.NdnPktType_Nack)
		} else {
			C.Packet_SetNdnPktType(pkt.c, C.NdnPktType_Interest)
		}
		*C.Packet_GetInterestHdr(pkt.c) = interest.c
	} else {
		data := netp.(*DataPkt)
		C.Packet_SetNdnPktType(pkt.c, C.NdnPktType_Data)
		*C.Packet_GetDataHdr(pkt.c) = data.c
	}
}
