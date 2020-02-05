package pktqueue

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

const BURST_SIZE_MAX = C.PKTQUEUE_BURST_SIZE_MAX

type Config struct {
	Capacity         int // Ring capacity, must be power of 2, default 131072 with delay/CoDel or 4096 without
	DequeueBurstSize int // dequeue burst size limit, default BURST_SIZE_MAX

	Delay        nnduration.Nanoseconds // if non-zero, enforce minimum delay, implies DisableCoDel
	DisableCoDel bool                   // if true, disable CoDel algorithm
	Target       nnduration.Nanoseconds // CoDel TARGET parameter, default 5ms
	Interval     nnduration.Nanoseconds // CoDel INTERVAL parameter, default 100ms
}

// A packet queue with simplified CoDel algorithm.
type PktQueue struct {
	c *C.PktQueue
}

func FromPtr(ptr unsafe.Pointer) (q PktQueue) {
	q.c = (*C.PktQueue)(ptr)
	return q
}

// Create PktQueue at given (*C.PktQueue) pointer.
func NewAt(ptr unsafe.Pointer, cfg Config, name string, socket dpdk.NumaSocket) (q PktQueue, e error) {
	q = FromPtr(ptr)
	*q.c = C.PktQueue{}

	capacity := 131072
	convertDuration := func(input nnduration.Nanoseconds, defaultMs time.Duration) C.TscDuration {
		d := input.Duration()
		if d == 0 {
			d = defaultMs * time.Millisecond
		}
		return C.TscDuration(dpdk.ToTscDuration(d))
	}
	switch {
	case cfg.Delay > 0:
		q.c.pop = C.PktQueue_PopOp(C.PktQueue_PopDelay)
		q.c.target = convertDuration(cfg.Delay, 0)
	case cfg.DisableCoDel:
		q.c.pop = C.PktQueue_PopOp(C.PktQueue_PopPlain)
		capacity = 4096
	default:
		q.c.pop = C.PktQueue_PopOp(C.PktQueue_PopCoDel)
		q.c.target = convertDuration(cfg.Target, 5)
		q.c.interval = convertDuration(cfg.Interval, 100)
	}
	if cfg.Capacity > 0 {
		capacity = cfg.Capacity
	}

	if r, e := dpdk.NewRing(name, capacity, socket, false, true); e != nil {
		return q, e
	} else {
		q.c.ring = (*C.struct_rte_ring)(r.GetPtr())
	}

	if cfg.DequeueBurstSize > 0 && cfg.DequeueBurstSize < BURST_SIZE_MAX {
		q.c.dequeueBurstSize = C.uint32_t(cfg.DequeueBurstSize)
	} else {
		q.c.dequeueBurstSize = BURST_SIZE_MAX
	}

	return q, nil
}

func (q PktQueue) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(q.c)
}

func (q PktQueue) GetRing() dpdk.Ring {
	return dpdk.RingFromPtr(unsafe.Pointer(q.c.ring))
}

func (q PktQueue) Close() error {
	return q.GetRing().Close()
}

func (q PktQueue) Push(pkts interface{}, now dpdk.TscTime) (nRej int) {
	ptr, count := dpdk.ParseCptrArray(pkts)
	return int(C.PktQueue_Push(q.c, (**C.struct_rte_mbuf)(ptr), C.uint(count), C.TscTime(now)))
}

func (q PktQueue) Pop(pkts interface{}, now dpdk.TscTime) (count int, drop bool) {
	ptr, count := dpdk.ParseCptrArray(pkts)
	res := C.PktQueue_Pop(q.c, (**C.struct_rte_mbuf)(ptr), C.uint(count), C.TscTime(now))
	return int(res.count), bool(res.drop)
}
