package dpdk

/*
#include <rte_config.h>
#include <rte_ring.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

type Ring struct {
	c *C.struct_rte_ring
	// DO NOT add other fields: *Ring is passed to C code as rte_ring**
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
	r.c = C.rte_ring_create(cName, C.uint(capacity), C.int(socket), flags)
	if r.c == nil {
		return r, GetErrno()
	}
	return r, nil
}

// Construct Ring from native *C.struct_rte_ring pointer.
func RingFromPtr(ptr unsafe.Pointer) Ring {
	return Ring{(*C.struct_rte_ring)(ptr)}
}

// Get native *C.struct_rte_ring pointer to use in other packages.
func (r Ring) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(r.c)
}

func (r Ring) Close() error {
	C.rte_ring_free(r.c)
	return nil
}

// Get ring capacity.
func (r Ring) GetCapacity() int {
	return int(C.rte_ring_get_capacity(r.c))
}

// Get used space.
func (r Ring) Count() int {
	return int(C.rte_ring_count(r.c))
}

func (r Ring) IsEmpty() bool {
	return r.Count() == 0
}

// Get free space.
func (r Ring) GetFreeSpace() int {
	return int(C.rte_ring_free_count(r.c))
}

func (r Ring) IsFull() bool {
	return r.GetFreeSpace() == 0
}

// Enqueue several objects on a ring.
// Return number of objects enqueued, and available ring space after operation.
func (r Ring) BurstEnqueue(objs interface{}) (nEnqueued int, freeSpace int) {
	ptr, count := ParseCptrArray(objs)
	var freeSpaceC C.uint
	res := C.rte_ring_enqueue_burst(r.c, (*unsafe.Pointer)(ptr), C.uint(count), &freeSpaceC)
	return int(res), int(freeSpaceC)
}

// Dequeue several objects on a ring, writing into slice of native pointers.
// Return number of objects dequeued, and remaining ring entries after operation.
func (r Ring) BurstDequeue(objs interface{}) (nDequeued int, nEntries int) {
	ptr, count := ParseCptrArray(objs)
	var nEntriesC C.uint
	res := C.rte_ring_dequeue_burst(r.c, (*unsafe.Pointer)(ptr), C.uint(count), &nEntriesC)
	return int(res), int(nEntriesC)
}

func init() {
	var r Ring
	if unsafe.Sizeof(r) != unsafe.Sizeof(r.c) {
		panic("sizeof dpdk.Ring differs from *C.struct_rte_ring")
	}
}
