// Package ringbuffer contains bindings of DPDK ring library.
package ringbuffer

/*
#include "../../csrc/core/common.h"
#include <rte_ring.h>
*/
import "C"
import (
	"math/bits"
	"unsafe"

	binutils "github.com/jfoster/binary-utilities"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/zyedidia/generic"
)

// Limits and defaults.
const (
	MinCapacity     = 4
	MaxCapacity     = (C.RTE_RING_SZ_MASK + 1) / 2
	DefaultCapacity = 256
)

// AlignCapacity adjusts Ring capacity to a power of two between minimum and maximum.
// Optional arguments: minimum capacity, default capacity, maximum capacity.
// Default capacity is used if input is zero.
func AlignCapacity(capacity int, opts ...int) int {
	min, dflt, max := MinCapacity, DefaultCapacity, MaxCapacity
	switch len(opts) {
	case 0:
	case 1:
		min, dflt = opts[0], opts[0]
	case 2:
		min, dflt = opts[0], opts[1]
	case 3:
		min, dflt, max = opts[0], opts[1], opts[2]
	default:
		panic("unexpected opts count")
	}
	if min <= 0 || dflt < min || dflt > max ||
		bits.OnesCount64(uint64(min)) != 1 || bits.OnesCount64(uint64(dflt)) != 1 || bits.OnesCount64(uint64(max)) != 1 {
		panic("invalid min, dflt, max")
	}

	if capacity <= 0 {
		capacity = dflt
	} else {
		capacity = int(binutils.NextPowerOfTwo(int64(capacity)))
	}
	return generic.Clamp(capacity, min, max)
}

// ProducerMode indicates ring producer synchronization mode.
type ProducerMode int

// Ring producer synchronization modes.
const (
	ProducerMulti  ProducerMode = 0
	ProducerSingle ProducerMode = C.RING_F_SP_ENQ
	ProducerRts    ProducerMode = C.RING_F_MP_RTS_ENQ
	ProducerHts    ProducerMode = C.RING_F_MP_HTS_ENQ
)

// ConsumerMode indicates ring consumer synchronization mode.
type ConsumerMode int

// Ring consumer synchronization modes.
const (
	ConsumerMulti  ConsumerMode = 0
	ConsumerSingle ConsumerMode = C.RING_F_SC_DEQ
	ConsumerRts    ConsumerMode = C.RING_F_MC_RTS_DEQ
	ConsumerHts    ConsumerMode = C.RING_F_MC_HTS_DEQ
)

// Ring represents a FIFO ring buffer.
type Ring C.struct_rte_ring

// FromPtr converts *C.struct_rte_ring pointer to Ring.
func FromPtr(ptr unsafe.Pointer) *Ring {
	return (*Ring)(ptr)
}

// Ptr returns *C.struct_rte_ring pointer.
func (r *Ring) Ptr() unsafe.Pointer {
	return unsafe.Pointer(r)
}

func (r *Ring) ptr() *C.struct_rte_ring {
	return (*C.struct_rte_ring)(r)
}

// Close releases the ring.
func (r *Ring) Close() error {
	C.rte_ring_free(r.ptr())
	return nil
}

func (r *Ring) String() string {
	return C.GoString(&r.name[0])
}

// Capacity returns ring capacity.
func (r *Ring) Capacity() int {
	return int(C.rte_ring_get_capacity(r.ptr()))
}

// CountAvailable returns free space.
func (r *Ring) CountAvailable() int {
	return int(C.rte_ring_free_count(r.ptr()))
}

// CountInUse returns used space.
func (r *Ring) CountInUse() int {
	return int(C.rte_ring_count(r.ptr()))
}

// New creates a Ring.
func New(capacity int, socket eal.NumaSocket, pm ProducerMode, cm ConsumerMode) (r *Ring, e error) {
	nameC := C.CString(eal.AllocObjectID("ringbuffer.Ring"))
	defer C.free(unsafe.Pointer(nameC))
	capacity = AlignCapacity(capacity)
	flags := C.uint(pm) | C.uint(cm)

	r = (*Ring)(C.rte_ring_create(nameC, C.uint(capacity), C.int(socket.ID()), flags))
	if r == nil {
		return nil, eal.GetErrno()
	}
	return r, nil
}

// Enqueue enqueues several objects.
func Enqueue[T any, A ~[]T](r *Ring, objs A) (nEnqueued int) {
	return int(C.rte_ring_enqueue_burst(r.ptr(), cptr.FirstPtr[unsafe.Pointer](objs), C.uint(len(objs)), nil))
}

// Dequeue dequeues several objects.
func Dequeue[T any, A ~[]T](r *Ring, objs A) (nDequeued int) {
	return int(C.rte_ring_dequeue_burst(r.ptr(), cptr.FirstPtr[unsafe.Pointer](objs), C.uint(len(objs)), nil))
}
