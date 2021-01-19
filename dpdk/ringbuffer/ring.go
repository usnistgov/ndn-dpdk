// Package ringbuffer contains bindings of DPDK ring library.
package ringbuffer

/*
#include "../../csrc/core/common.h"
#include <rte_ring.h>
*/
import "C"
import (
	"unsafe"

	binutils "github.com/jfoster/binary-utilities"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

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

// Ring consumer synchronzation modes.
const (
	ConsumerMulti  ConsumerMode = 0
	ConsumerSingle ConsumerMode = C.RING_F_SC_DEQ
	ConsumerRts    ConsumerMode = C.RING_F_MC_RTS_DEQ
	ConsumerCts    ConsumerMode = C.RING_F_MC_HTS_DEQ
)

// Ring represents a FIFO ring buffer.
type Ring C.struct_rte_ring

// New creates a Ring.
func New(capacity int, socket eal.NumaSocket,
	producerMode ProducerMode, consumerMode ConsumerMode) (r *Ring, e error) {
	nameC := C.CString(eal.AllocObjectID("ringbuffer.Ring"))
	defer C.free(unsafe.Pointer(nameC))
	capacity = AlignCapacity(capacity, 4, 64)
	flags := C.uint(producerMode) | C.uint(consumerMode)

	ringC := C.rte_ring_create(nameC, C.uint(capacity), C.int(socket.ID()), flags)
	if ringC == nil {
		return nil, eal.GetErrno()
	}
	return (*Ring)(ringC), nil
}

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
	return C.GoString(&r.ptr().name[0])
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

// Enqueue enqueues several objects on a ring.
// objs should be a slice of C void* pointers.
func (r *Ring) Enqueue(objs interface{}) (nEnqueued int) {
	ptr, count := cptr.ParseCptrArray(objs)
	return int(C.rte_ring_enqueue_burst(r.ptr(), (*unsafe.Pointer)(ptr), C.uint(count), nil))
}

// Dequeue dequeues several objects from a ring.
// objs should be a slice of C void* pointers.
func (r *Ring) Dequeue(objs interface{}) (nDequeued int) {
	ptr, count := cptr.ParseCptrArray(objs)
	return int(C.rte_ring_dequeue_burst(r.ptr(), (*unsafe.Pointer)(ptr), C.uint(count), nil))
}

// AlignCapacity returns an acceptable capacity for Ring.
// It takes up to three parameters:
//   capacity: input capacity
//   min: minimum capacity; default is 64.
//   dflt: default capacity, if input is less than minimum; default is same as min.
//
// If input capacity is less than minimum, use dflt. Then, adjust to next power of 2.
func AlignCapacity(capacity int, opts ...int) int {
	var min, dflt int
	switch len(opts) {
	case 0:
		min, dflt = 64, 64
	case 1:
		min, dflt = opts[0], opts[0]
	case 2:
		min, dflt = opts[0], opts[1]
	default:
		panic("opts")
	}

	if capacity < min {
		capacity = dflt
	}
	return int(binutils.NextPowerOfTwo(int64(capacity)))
}
