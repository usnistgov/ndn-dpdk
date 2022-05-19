package iface

/*
#include "../csrc/iface/pktqueue.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
)

// PktQueueConfig contains PktQueue configuration.
type PktQueueConfig struct {
	// Ring capacity, must be power of 2, default 131072 with delay/CoDel or 4096 without
	Capacity int `json:"capacity,omitempty"`
	// dequeue burst size limit, default MaxBurstSize
	DequeueBurstSize int `json:"dequeueBurstSize,omitempty"`

	// if non-zero, enforce minimum delay, implies DisableCoDel
	Delay nnduration.Nanoseconds `json:"delay,omitempty"`
	// if true, disable CoDel algorithm
	DisableCoDel bool `json:"disableCoDel,omitempty"`
	// CoDel TARGET parameter, default 5ms
	Target nnduration.Nanoseconds `json:"target,omitempty"`
	// CoDel INTERVAL parameter, default 100ms
	Interval nnduration.Nanoseconds `json:"interval,omitempty"`
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
	capacity := 131072
	switch {
	case cfg.Delay > 0:
		q.pop = C.PktQueuePopActDelay
		q.target = C.TscDuration(eal.ToTscDuration(cfg.Delay.Duration()))
	case cfg.DisableCoDel:
		q.pop = C.PktQueuePopActPlain
		capacity = 4096
	default:
		q.pop = C.PktQueuePopActCoDel
		q.target = C.TscDuration(eal.ToTscDuration(cfg.Target.DurationOr(nnduration.Nanoseconds(5 * time.Millisecond))))
		q.interval = C.TscDuration(eal.ToTscDuration(cfg.Interval.DurationOr(nnduration.Nanoseconds(100 * time.Millisecond))))
	}
	if cfg.Capacity > 0 {
		capacity = cfg.Capacity
	}

	ring, e := ringbuffer.New(capacity, socket, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle)
	if e != nil {
		return e
	}
	q.ring = (*C.struct_rte_ring)(ring.Ptr())

	if cfg.DequeueBurstSize > 0 && cfg.DequeueBurstSize < MaxBurstSize {
		q.dequeueBurstSize = C.uint32_t(cfg.DequeueBurstSize)
	} else {
		q.dequeueBurstSize = MaxBurstSize
	}

	return nil
}

// Ring provides access to the internal ring.
func (q *PktQueue) Ring() *ringbuffer.Ring {
	return ringbuffer.FromPtr(unsafe.Pointer(q.ring))
}

// Close drains and deallocates the PktQueue.
// It will not free *C.PktQueue itself.
func (q *PktQueue) Close() error {
	ring := q.Ring()
	if ring == nil {
		return nil
	}
	q.ring = nil

	vec := make(pktmbuf.Vector, MaxBurstSize)
	for {
		n := ringbuffer.Dequeue(ring, vec)
		if n == 0 {
			break
		}
		vec[:n].Close()
	}
	return ring.Close()
}

// Push enqueues a slice of packets.
// Timestamps must have been assigned on the packets.
// Caller must free rejected packets.
func (q *PktQueue) Push(vec pktmbuf.Vector) (nRej int) {
	return int(C.PktQueue_Push(q.ptr(), cptr.FirstPtr[*C.struct_rte_mbuf](vec), C.uint32_t(len(vec))))
}

// Pop dequeues a slice of packets.
func (q *PktQueue) Pop(vec pktmbuf.Vector, now eal.TscTime) (count int, drop bool) {
	res := C.PktQueue_Pop(q.ptr(), cptr.FirstPtr[*C.struct_rte_mbuf](vec), C.uint32_t(len(vec)), C.TscTime(now))
	return int(res.count), bool(res.drop)
}

// PktQueueCounters contains PktQueue counters.
type PktQueueCounters struct {
	NDrops uint64 `json:"nDrops"`
}

// Counters reads counters.
func (q *PktQueue) Counters() (cnt PktQueueCounters) {
	cnt.NDrops = uint64(q.nDrops)
	return cnt
}
