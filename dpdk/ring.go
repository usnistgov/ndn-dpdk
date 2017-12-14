package dpdk

/*
#include <rte_config.h>
#include <rte_ring.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

type Ring struct {
	ptr *C.struct_rte_ring
}

func NewRing(name string, capacity int, socket NumaSocket,
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

func (r Ring) Count() int {
	return int(C.rte_ring_count(r.ptr))
}

func (r Ring) IsEmpty() bool {
	return r.Count() == 0
}

func (r Ring) GetFreeSpace() int {
	return int(C.rte_ring_free_count(r.ptr))
}

func (r Ring) IsFull() bool {
	return r.GetFreeSpace() == 0
}

// Enqueue several objects on a ring.
// Return number of objects enqueued, and available ring space after operation.
func (r Ring) BurstEnqueue(objs []unsafe.Pointer) (int, int) {
	var freeSpace C.uint
	res := C.rte_ring_enqueue_burst(r.ptr, &objs[0], C.uint(len(objs)), &freeSpace)
	return int(res), int(freeSpace)
}

// Dequeue several objects on a ring, writing into slice of native pointers.
// Return number of objects dequeued, and remaining ring entries after operation.
func (r Ring) BurstDequeue(objs []unsafe.Pointer) (int, int) {
	var nEntries C.uint
	res := C.rte_ring_dequeue_burst(r.ptr, &objs[0], C.uint(len(objs)), &nEntries)
	return int(res), int(nEntries)
}
