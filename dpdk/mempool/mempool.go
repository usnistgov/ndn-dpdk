package mempool

/*
#include "../../core/common.h"
#include <rte_mempool.h>
*/
import "C"
import (
	"errors"
	"unsafe"

	"ndn-dpdk/core/cptr"
	"ndn-dpdk/dpdk/eal"
)

// ComputeCacheSize calculates the appropriate cache size for given mempool capacity.
func ComputeCacheSize(capacity int) int {
	max := C.RTE_MEMPOOL_CACHE_MAX_SIZE
	if capacity/16 < max {
		return capacity / 16
	}
	min := max / 4
	for i := max; i >= min; i-- {
		if capacity%i == 0 {
			return i
		}
	}
	return max
}

// Mempool represents a DPDK memory pool for generic objects.
type Mempool C.struct_rte_mempool

// New creates a Mempool.
func New(name string, capacity int, elementSize int, socket eal.NumaSocket) (mp *Mempool, e error) {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))

	mempoolC := C.rte_mempool_create(nameC, C.uint(capacity), C.uint(elementSize),
		C.uint(ComputeCacheSize(capacity)), 0, nil, nil, nil, nil, C.int(socket.ID()), 0)
	if mempoolC == nil {
		return nil, eal.GetErrno()
	}
	return (*Mempool)(mempoolC), nil
}

// FromPtr converts *C.struct_rte_mempool pointer to Mempool.
func FromPtr(ptr unsafe.Pointer) *Mempool {
	return (*Mempool)(ptr)
}

// GetPtr returns *C.struct_rte_mempool pointer.
func (mp *Mempool) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(mp)
}

func (mp *Mempool) getPtr() *C.struct_rte_mempool {
	return (*C.struct_rte_mempool)(mp)
}

// Close releases the mempool.
func (mp *Mempool) Close() error {
	C.rte_mempool_free(mp.getPtr())
	return nil
}

// SizeofElement returns element size.
func (mp *Mempool) SizeofElement() int {
	return int(mp.getPtr().elt_size)
}

// CountAvailable returns number of available objects.
func (mp *Mempool) CountAvailable() int {
	return int(C.rte_mempool_avail_count(mp.getPtr()))
}

// CountInUse returns number of allocated objects.
func (mp *Mempool) CountInUse() int {
	return int(C.rte_mempool_in_use_count(mp.getPtr()))
}

// Alloc allocates several objects.
// objs should be a slice of C void* pointers.
func (mp *Mempool) Alloc(objs interface{}) error {
	ptr, count := cptr.ParseCptrArray(objs)
	if count == 0 {
		return nil
	}
	res := C.rte_mempool_get_bulk(mp.getPtr(), (*unsafe.Pointer)(ptr), C.uint(count))
	if res != 0 {
		return errors.New("mbuf allocation failed")
	}
	return nil
}

// Free releases several objects.
// objs should be a slice of C void* pointers.
func (mp *Mempool) Free(objs interface{}) {
	ptr, count := cptr.ParseCptrArray(objs)
	if count == 0 {
		return
	}
	C.rte_mempool_put_bulk(mp.getPtr(), (*unsafe.Pointer)(ptr), C.uint(count))
}
