package iface

/*
#include "../csrc/iface/pktqueue.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
)

// PktQueueConfig contains PktQueue configuration.
type PktQueueConfig struct {
	Capacity         int // Ring capacity, must be power of 2, default 131072 with delay/CoDel or 4096 without
	DequeueBurstSize int // dequeue burst size limit, default iface.MaxBurstSize

	Delay        nnduration.Nanoseconds // if non-zero, enforce minimum delay, implies DisableCoDel
	DisableCoDel bool                   // if true, disable CoDel algorithm
	Target       nnduration.Nanoseconds // CoDel TARGET parameter, default 5ms
	Interval     nnduration.Nanoseconds // CoDel INTERVAL parameter, default 100ms
}

// PktQueue is a packet queue with simplified CoDel algorithm.
type PktQueue C.PktQueue

// PktQueueFromPtr converts *C.PktQueue to PktQueue.
func PktQueueFromPtr(ptr unsafe.Pointer) (q *PktQueue) {
	return (*PktQueue)(ptr)
}

// Ptr return *C.PktQueue pointer.
func (q *PktQueue) Ptr() unsafe.Pointer {
	return unsafe.Pointer(q)
}

func (q *PktQueue) ptr() *C.PktQueue {
	return (*C.PktQueue)(q)
}

// Init initializes PktQueue.
func (q *PktQueue) Init(cfg PktQueueConfig, socket eal.NumaSocket) error {
	c := q.ptr()

	capacity := 131072
	convertDuration := func(input nnduration.Nanoseconds, defaultMs time.Duration) C.TscDuration {
		d := input.Duration()
		if d == 0 {
			d = defaultMs * time.Millisecond
		}
		return C.TscDuration(eal.ToTscDuration(d))
	}
	switch {
	case cfg.Delay > 0:
		c.pop = C.PktQueue_PopOp(C.PktQueue_PopDelay)
		c.target = convertDuration(cfg.Delay, 0)
	case cfg.DisableCoDel:
		c.pop = C.PktQueue_PopOp(C.PktQueue_PopPlain)
		capacity = 4096
	default:
		c.pop = C.PktQueue_PopOp(C.PktQueue_PopCoDel)
		c.target = convertDuration(cfg.Target, 5)
		c.interval = convertDuration(cfg.Interval, 100)
	}
	if cfg.Capacity > 0 {
		capacity = cfg.Capacity
	}

	ring, e := ringbuffer.New(capacity, socket, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle)
	if e != nil {
		return e
	}
	c.ring = (*C.struct_rte_ring)(ring.Ptr())

	if cfg.DequeueBurstSize > 0 && cfg.DequeueBurstSize < MaxBurstSize {
		c.dequeueBurstSize = C.uint32_t(cfg.DequeueBurstSize)
	} else {
		c.dequeueBurstSize = MaxBurstSize
	}

	return nil
}

// Close deallocates the PktQueue.
func (q *PktQueue) Close() error {
	ring := ringbuffer.FromPtr(unsafe.Pointer(q.ptr().ring))
	return ring.Close()
}

// Push enqueues a slice of packets.
func (q *PktQueue) Push(vec pktmbuf.Vector, now eal.TscTime) (nRej int) {
	return int(C.PktQueue_Push(q.ptr(), (**C.struct_rte_mbuf)(vec.Ptr()), C.uint(len(vec)), C.TscTime(now)))
}

// Pop dequeues a slice of packets.
func (q *PktQueue) Pop(vec pktmbuf.Vector, now eal.TscTime) (count int, drop bool) {
	res := C.PktQueue_Pop(q.ptr(), (**C.struct_rte_mbuf)(vec.Ptr()), C.uint(len(vec)), C.TscTime(now))
	return int(res.count), bool(res.drop)
}
