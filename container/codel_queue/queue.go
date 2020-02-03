package codel_queue

/*
#include "queue.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"ndn-dpdk/core/nnduration"
	"ndn-dpdk/dpdk"
)

const BURST_SIZE_MAX = C.CODELQUEUE_BURST_SIZE_MAX

type Config struct {
	Target   nnduration.Nanoseconds // CoDel TARGET parameter, default 5ms
	Interval nnduration.Nanoseconds // CoDel INTERVAL parameter, default 100ms

	DequeueBurstSize int // dequeue burst size limit, default BURST_SIZE_MAX
}

type CoDelQueue struct {
	c *C.CoDelQueue
}

func FromPtr(ptr unsafe.Pointer) (q CoDelQueue) {
	q.c = (*C.CoDelQueue)(ptr)
	return q
}

func NewAt(ptr unsafe.Pointer, cfg Config, r dpdk.Ring) (q CoDelQueue) {
	var target, interval time.Duration
	if target = cfg.Target.Duration(); target == 0 {
		target = 5 * time.Millisecond
	}
	if interval = cfg.Interval.Duration(); interval == 0 {
		interval = 100 * time.Millisecond
	}

	q = FromPtr(ptr)
	*q.c = C.CoDelQueue{}
	q.c.ring = (*C.struct_rte_ring)(r.GetPtr())
	q.c.target = C.TscDuration(dpdk.ToTscDuration(target))
	q.c.interval = C.TscDuration(dpdk.ToTscDuration(interval))
	if q.c.dequeueBurstSize = C.uint32_t(cfg.DequeueBurstSize); q.c.dequeueBurstSize == 0 || q.c.dequeueBurstSize > BURST_SIZE_MAX {
		q.c.dequeueBurstSize = BURST_SIZE_MAX
	}
	return q
}

func (q CoDelQueue) GetRing() dpdk.Ring {
	return dpdk.RingFromPtr(unsafe.Pointer(q.c.ring))
}

func (q CoDelQueue) Push(pkts interface{}, now dpdk.TscTime) (nRej int) {
	ptr, count := dpdk.ParseCptrArray(pkts)
	return int(C.CoDelQueue_Push(q.c, (**C.struct_rte_mbuf)(ptr), C.uint(count), C.TscTime(now)))
}

func (q CoDelQueue) Pop(pkts interface{}, now dpdk.TscTime) (count int, drop bool) {
	ptr, count := dpdk.ParseCptrArray(pkts)
	res := C.CoDelQueue_Pop(q.c, (**C.struct_rte_mbuf)(ptr), C.uint(count), C.TscTime(now))
	return int(res.count), bool(res.drop)
}
