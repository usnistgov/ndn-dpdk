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

type NdnNetType int

const (
	NdnNetType_None     NdnNetType = C.NdnNetType_None
	NdnNetType_Interest            = C.NdnNetType_Interest
	NdnNetType_Data                = C.NdnNetType_Data
	NdnNetType_Nack                = C.NdnNetType_Nack
)

type Packet struct {
	dpdk.Packet
}

func (pkt Packet) getPtr() *C.struct_rte_mbuf {
	return (*C.struct_rte_mbuf)(pkt.GetPtr())
}

func (pkt Packet) GetNetType() NdnNetType {
	return NdnNetType(C.Packet_GetNdnNetType(pkt.getPtr()))
}

func (pkt Packet) AsInterest() *ndn.InterestPkt {
	return (*ndn.InterestPkt)(unsafe.Pointer(C.Packet_GetInterestHdr(pkt.getPtr())))
}

func (pkt Packet) AsData() *ndn.DataPkt {
	return (*ndn.DataPkt)(unsafe.Pointer(C.Packet_GetDataHdr(pkt.getPtr())))
}
