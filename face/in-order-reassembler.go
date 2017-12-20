package face

/*
#include "in-order-reassembler.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

type InOrderReassembler struct {
	c C.InOrderReassembler
}

func (r *InOrderReassembler) Receive(pkt Packet) Packet {
	res := C.InOrderReassembler_Receive(&r.c, pkt.getPtr())
	return Packet{dpdk.MbufFromPtr(unsafe.Pointer(res)).AsPacket()}
}

type InOrderReassemblerCounters struct {
	NAccepted   uint64
	NOutOfOrder uint64
	NDelivered  uint64
}

func (r *InOrderReassembler) GetCounters() InOrderReassemblerCounters {
	return InOrderReassemblerCounters{
		NAccepted:   uint64(r.c.nAccepted),
		NOutOfOrder: uint64(r.c.nOutOfOrder),
		NDelivered:  uint64(r.c.nDelivered),
	}
}
