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
	// DO NOT add other fields: *Packet is passed to C code as Packet**
}

// Construct Packet from *C.Packet.
// This function can accept nil pointer.
func PacketFromPtr(ptr unsafe.Pointer) (pkt Packet) {
	if ptr != nil {
		pkt.c = C.Packet_FromMbuf((*C.struct_rte_mbuf)(ptr))
	}
	return pkt
}

func PacketFromDpdk(m dpdk.Packet) (pkt Packet) {
	return PacketFromPtr(m.GetPtr())
}

func (pkt Packet) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(pkt.c)
}

func (pkt Packet) AsDpdkPacket() dpdk.Packet {
	return dpdk.MbufFromPtr(unsafe.Pointer(pkt.c)).AsPacket()
}

func (pkt Packet) GetL2Type() L2PktType {
	return L2PktType(C.Packet_GetL2PktType(pkt.c))
}

func (pkt Packet) GetL3Type() L3PktType {
	return L3PktType(C.Packet_GetL3PktType(pkt.c))
}

func (pkt Packet) GetLpHdr() *LpHeader {
	return (*LpHeader)(unsafe.Pointer(C.Packet_GetLpHdr(pkt.c)))
}

func (pkt Packet) GetLpL3() *LpL3 {
	return (*LpL3)(unsafe.Pointer(C.Packet_GetLpL3Hdr(pkt.c)))
}

func (pkt Packet) AsInterest() *Interest {
	return &Interest{pkt, C.Packet_GetInterestHdr(pkt.c)}
}

func (pkt Packet) AsData() *Data {
	return &Data{pkt, C.Packet_GetDataHdr(pkt.c)}
}

func (pkt Packet) AsNack() *Nack {
	return &Nack{pkt, C.Packet_GetNackHdr(pkt.c)}
}

func (pkt Packet) String() string {
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

func (pkt Packet) ParseL2() error {
	res := NdnError(C.Packet_ParseL2(pkt.c))
	if res != NdnError_OK {
		return res
	}
	return nil
}

func (pkt Packet) ParseL3(nameMp dpdk.PktmbufPool) error {
	res := NdnError(C.Packet_ParseL3(pkt.c, (*C.struct_rte_mempool)(nameMp.GetPtr())))
	if res != NdnError_OK {
		return res
	}
	return nil
}

// L3 packet interface type that allows conversion to Packet.
type IL3Packet interface {
	GetPacket() Packet
}

func init() {
	var pkt Packet
	if unsafe.Sizeof(pkt) != unsafe.Sizeof(pkt.c) {
		panic("sizeof ndn.Packet differs from *C.Packet")
	}
}
