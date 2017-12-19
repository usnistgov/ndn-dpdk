package face

/*
#include "packet.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
)

type NdnPktType int

const (
	NdnPktType_None     NdnPktType = C.NdnPktType_None
	NdnPktType_Interest            = C.NdnPktType_Interest
	NdnPktType_Data                = C.NdnPktType_Data
	NdnPktType_Nack                = C.NdnPktType_Nack
)

type Packet struct {
	dpdk.Packet
}

func (pkt Packet) getPtr() *C.struct_rte_mbuf {
	return (*C.struct_rte_mbuf)(pkt.GetPtr())
}

func (pkt Packet) GetNetType() NdnPktType {
	return NdnPktType(C.Packet_GetNdnPktType(pkt.getPtr()))
}

func (pkt Packet) AsInterest() *ndn.InterestPkt {
	return (*ndn.InterestPkt)(unsafe.Pointer(C.Packet_GetInterestHdr(pkt.getPtr())))
}

func (pkt Packet) AsData() *ndn.DataPkt {
	return (*ndn.DataPkt)(unsafe.Pointer(C.Packet_GetDataHdr(pkt.getPtr())))
}
