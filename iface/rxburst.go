package iface

/*
#include "../csrc/iface/rxburst.h"

void
c_FaceRxBurst_SetFrame(FaceRxBurst* burst, int i, struct rte_mbuf* frame)
{
	FaceRxBurst_GetScratch(burst)[i] = frame;
}
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// A burst of received packets.
type RxBurst struct {
	c *C.FaceRxBurst
}

func NewRxBurst(capacity int) (burst RxBurst) {
	burst.c = C.FaceRxBurst_New(C.uint16_t(capacity))
	return burst
}

func (burst RxBurst) Ptr() unsafe.Pointer {
	return unsafe.Pointer(burst.c)
}

func (burst RxBurst) Close() error {
	C.FaceRxBurst_Close(burst.c)
	return nil
}

func (burst RxBurst) Capacity() int {
	return int(burst.c.capacity)
}

func (burst RxBurst) ListInterests() (list []*ndni.Interest) {
	list = make([]*ndni.Interest, int(burst.c.nInterests))
	for i := range list {
		npkt := ndni.PacketFromPtr(unsafe.Pointer(C.FaceRxBurst_GetInterest(burst.c, C.uint16_t(i))))
		list[i] = npkt.AsInterest()
	}
	return list
}

func (burst RxBurst) ListData() (list []*ndni.Data) {
	list = make([]*ndni.Data, int(burst.c.nData))
	for i := range list {
		npkt := ndni.PacketFromPtr(unsafe.Pointer(C.FaceRxBurst_GetData(burst.c, C.uint16_t(i))))
		list[i] = npkt.AsData()
	}
	return list
}

func (burst RxBurst) ListNacks() (list []*ndni.Nack) {
	list = make([]*ndni.Nack, int(burst.c.nNacks))
	for i := range list {
		npkt := ndni.PacketFromPtr(unsafe.Pointer(C.FaceRxBurst_GetNack(burst.c, C.uint16_t(i))))
		list[i] = npkt.AsNack()
	}
	return list
}

// Put received frame in scratch space.
func (burst RxBurst) SetFrame(i int, frame *pktmbuf.Packet) {
	C.c_FaceRxBurst_SetFrame(burst.c, C.int(i), (*C.struct_rte_mbuf)(frame.Ptr()))
}
