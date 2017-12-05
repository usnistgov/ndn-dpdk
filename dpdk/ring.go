package dpdk

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk

#include <rte_config.h>
#include <rte_ring.h>
#include <stdlib.h>
*/
import "C"
import (
	// "errors"
	"unsafe"
)

// An array of void* on C memory.
type RingObjTable struct {
	table    uintptr
	capacity uint
}

const voidPtrSize = unsafe.Sizeof(unsafe.Pointer(nil))

func NewRingObjTable(capacity uint) *RingObjTable {
	ot := new(RingObjTable)
	ot.table = uintptr(C.calloc(C.size_t(capacity), C.size_t(voidPtrSize)))
	ot.capacity = capacity
	return ot
}

func (ot *RingObjTable) Close() {
	C.free(unsafe.Pointer(ot.table))
}

func (ot *RingObjTable) GetCapacity() uint {
	return ot.capacity
}

// Get underlying pointer as void**
func (ot *RingObjTable) getPointer() *unsafe.Pointer {
	return (*unsafe.Pointer)(unsafe.Pointer(ot.table))
}

func (ot *RingObjTable) Get(i uint) unsafe.Pointer {
	ptr := unsafe.Pointer(ot.table + voidPtrSize*uintptr(i))
	return *(*unsafe.Pointer)(ptr)
}

func (ot *RingObjTable) Set(i uint, v unsafe.Pointer) {
	ptr := unsafe.Pointer(ot.table + voidPtrSize*uintptr(i))
	*(*unsafe.Pointer)(ptr) = v
}

type Ring struct {
	ptr *C.struct_rte_ring
}

func NewRing(name string, capacity uint, socket NumaSocket,
	isSingleProducer bool, isSingleConsumer bool) (Ring, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var flags C.uint
	if isSingleProducer {
		flags = flags | C.RING_F_SP_ENQ
	}
	if isSingleConsumer {
		flags = flags | C.RING_F_SC_DEQ
	}

	var r Ring
	r.ptr = C.rte_ring_create(cName, C.uint(capacity), C.int(socket), flags)
	if r.ptr == nil {
		return r, GetErrno()
	}
	return r, nil
}

func (r Ring) Close() {
	C.rte_ring_free(r.ptr)
}

func (r Ring) Count() uint {
	return uint(C.rte_ring_count(r.ptr))
}

func (r Ring) IsEmpty() bool {
	return r.Count() == 0
}

func (r Ring) GetFreeSpace() uint {
	return uint(C.rte_ring_free_count(r.ptr))
}

func (r Ring) IsFull() bool {
	return r.GetFreeSpace() == 0
}

// Enqueue several objects on a ring.
// Return number of objects enqueued, and available ring space after operation.
func (r Ring) BurstEnqueue(ot *RingObjTable) (uint, uint) {
	var freeSpace C.uint
	res := C.rte_ring_enqueue_burst(r.ptr, ot.getPointer(), C.uint(ot.GetCapacity()), &freeSpace)
	return uint(res), uint(freeSpace)
}

// Dequeue several objects on a ring.
// Return number of objects dequeued, and remaining ring entries after operation.
func (r Ring) BurstDequeue(ot *RingObjTable) (uint, uint) {
	var nEntries C.uint
	res := C.rte_ring_dequeue_burst(r.ptr, ot.getPointer(), C.uint(ot.GetCapacity()), &nEntries)
	return uint(res), uint(nEntries)
}
