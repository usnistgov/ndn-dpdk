package ndn

/*
#include "packet.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

// Get size of PacketPriv structure.
// PktmbufPool's privSize must be no less than this size.
func SizeofPacketPriv() uint16 {
	return uint16(C.sizeof_PacketPriv)
}

type NdnPktType int

const (
	NdnPktType_None     NdnPktType = C.NdnPktType_None
	NdnPktType_Interest            = C.NdnPktType_Interest
	NdnPktType_Data                = C.NdnPktType_Data
	NdnPktType_Nack                = C.NdnPktType_Nack
)

// NDN network layer packet with parsed LP and Interest/Data headers.
type Packet struct {
	dpdk.Packet
}

// Construct Packet from *C.struct_rte_mbuf pointing to first segment.
// This function can accept nil pointer.
func PacketFromPtr(ptr unsafe.Pointer) Packet {
	return Packet{dpdk.MbufFromPtr(ptr).AsPacket()}
}

func (pkt Packet) getPtr() *C.struct_rte_mbuf {
	return (*C.struct_rte_mbuf)(pkt.GetPtr())
}

func (pkt Packet) GetLpHdr() *LpPkt {
	return (*LpPkt)(unsafe.Pointer(C.Packet_GetLpHdr(pkt.getPtr())))
}

func (pkt Packet) SetLpHdr(lpp LpPkt) {
	lpp1 := pkt.GetLpHdr()
	*lpp1 = lpp
}

func (pkt Packet) GetNetType() NdnPktType {
	return NdnPktType(C.Packet_GetNdnPktType(pkt.getPtr()))
}

func (pkt Packet) AsInterest() *InterestPkt {
	return (*InterestPkt)(unsafe.Pointer(C.Packet_GetInterestHdr(pkt.getPtr())))
}

func (pkt Packet) AsData() *DataPkt {
	return (*DataPkt)(unsafe.Pointer(C.Packet_GetDataHdr(pkt.getPtr())))
}
