package ringbuffer

/*
#include "../../csrc/core/common.h"
#include <rte_ring.h>
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/core/cptr"
	"ndn-dpdk/dpdk/eal"
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
func New(name string, capacity int, socket eal.NumaSocket,
	producerMode ProducerMode, consumerMode ConsumerMode) (r *Ring, e error) {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	capacity = AlignCapacity(capacity)
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

// GetPtr returns *C.struct_rte_ring pointer.
func (r *Ring) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(r)
}

func (r *Ring) getPtr() *C.struct_rte_ring {
	return (*C.struct_rte_ring)(r)
}

// Close releases the ring.
func (r *Ring) Close() error {
	C.rte_ring_free(r.getPtr())
	return nil
}

// GetCapacity returns ring capacity.
func (r *Ring) GetCapacity() int {
	return int(C.rte_ring_get_capacity(r.getPtr()))
}

// CountAvailable returns free space.
func (r *Ring) CountAvailable() int {
	return int(C.rte_ring_free_count(r.getPtr()))
}

// CountInUse returns used space.
func (r *Ring) CountInUse() int {
	return int(C.rte_ring_count(r.getPtr()))
}

// Enqueue enqueues several objects on a ring.
// objs should be a slice of C void* pointers.
func (r *Ring) Enqueue(objs interface{}) (nEnqueued int) {
	ptr, count := cptr.ParseCptrArray(objs)
	return int(C.rte_ring_enqueue_burst(r.getPtr(), (*unsafe.Pointer)(ptr), C.uint(count), nil))
}

// Dequeue dequeues several objects from a ring.
// objs should be a slice of C void* pointers.
func (r *Ring) Dequeue(objs interface{}) (nDequeued int) {
	ptr, count := cptr.ParseCptrArray(objs)
	return int(C.rte_ring_dequeue_burst(r.getPtr(), (*unsafe.Pointer)(ptr), C.uint(count), nil))
}

// AlignCapacity returns an acceptable capacity for Ring.
// If input capacity non-positive, returns 64.
// Otherwise, returns next power of 2.
func AlignCapacity(capacity int) int {
	if capacity <= 0 {
		return 64
	}
	return int(C.rte_align64pow2(C.uint64_t(capacity)))
}
