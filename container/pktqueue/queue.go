package pktqueue

/*
#include "../../csrc/pktqueue/queue.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
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
type PktQueue C.PktQueue

func FromPtr(ptr unsafe.Pointer) (q *PktQueue) {
	return (*PktQueue)(ptr)
}

// Create PktQueue at given (*C.PktQueue) pointer.
func NewAt(ptr unsafe.Pointer, cfg Config, name string, socket eal.NumaSocket) (q *PktQueue, e error) {
	qC := (*C.PktQueue)(ptr)

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
		qC.pop = C.PktQueue_PopOp(C.PktQueue_PopDelay)
		qC.target = convertDuration(cfg.Delay, 0)
	case cfg.DisableCoDel:
		qC.pop = C.PktQueue_PopOp(C.PktQueue_PopPlain)
		capacity = 4096
	default:
		qC.pop = C.PktQueue_PopOp(C.PktQueue_PopCoDel)
		qC.target = convertDuration(cfg.Target, 5)
		qC.interval = convertDuration(cfg.Interval, 100)
	}
	if cfg.Capacity > 0 {
		capacity = cfg.Capacity
	}

	if r, e := ringbuffer.New(name, capacity, socket, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle); e != nil {
		return q, e
	} else {
		qC.ring = (*C.struct_rte_ring)(r.Ptr())
	}

	if cfg.DequeueBurstSize > 0 && cfg.DequeueBurstSize < BURST_SIZE_MAX {
		qC.dequeueBurstSize = C.uint32_t(cfg.DequeueBurstSize)
	} else {
		qC.dequeueBurstSize = BURST_SIZE_MAX
	}

	return FromPtr(ptr), nil
}

func (q *PktQueue) Ptr() unsafe.Pointer {
	return unsafe.Pointer(q)
}

func (q *PktQueue) ptr() *C.PktQueue {
	return (*C.PktQueue)(q)
}

func (q *PktQueue) Close() error {
	ring := ringbuffer.FromPtr(unsafe.Pointer(q.ptr().ring))
	return ring.Close()
}

func (q *PktQueue) Push(pkts interface{}, now eal.TscTime) (nRej int) {
	ptr, count := cptr.ParseCptrArray(pkts)
	return int(C.PktQueue_Push(q.ptr(), (**C.struct_rte_mbuf)(ptr), C.uint(count), C.TscTime(now)))
}

func (q *PktQueue) Pop(pkts interface{}, now eal.TscTime) (count int, drop bool) {
	ptr, count := cptr.ParseCptrArray(pkts)
	res := C.PktQueue_Pop(q.ptr(), (**C.struct_rte_mbuf)(ptr), C.uint(count), C.TscTime(now))
	return int(res.count), bool(res.drop)
}
