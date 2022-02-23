// Package mempool contains bindings of DPDK memory pool.
package mempool

/*
#include "../../csrc/core/common.h"
#include <rte_mempool.h>
*/
import "C"
import (
	"errors"
	"math/bits"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// ComputeOptimumCapacity adjusts mempool capacity to be a power of two minus one, if near.
func ComputeOptimumCapacity(capacity int) int {
	if bits.OnesCount64(uint64(capacity)) == 1 {
		capacity--
	}
	return capacity
}

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

// Config contains Mempool configuration.
type Config struct {
	Capacity    int
	ElementSize int
	PrivSize    int
	Socket      eal.NumaSocket

	SingleProducer bool
	SingleConsumer bool
}

// Mempool represents a DPDK memory pool for generic objects.
type Mempool C.struct_rte_mempool

// Ptr returns *C.struct_rte_mempool pointer.
func (mp *Mempool) Ptr() unsafe.Pointer {
	return unsafe.Pointer(mp)
}

func (mp *Mempool) ptr() *C.struct_rte_mempool {
	return (*C.struct_rte_mempool)(mp)
}

// Close releases the mempool.
func (mp *Mempool) Close() error {
	C.rte_mempool_free(mp.ptr())
	return nil
}

func (mp *Mempool) String() string {
	return C.GoString(&mp.name[0])
}

// SizeofElement returns element size.
func (mp *Mempool) SizeofElement() int {
	return int(mp.elt_size)
}

// CountAvailable returns number of available objects.
func (mp *Mempool) CountAvailable() int {
	return int(C.rte_mempool_avail_count(mp.ptr()))
}

// CountInUse returns number of allocated objects.
func (mp *Mempool) CountInUse() int {
	return int(C.rte_mempool_in_use_count(mp.ptr()))
}

// Alloc allocates several objects.
// objs should be a slice of C void* pointers.
func (mp *Mempool) Alloc(objs interface{}) error {
	ptr, count := cptr.ParseCptrArray(objs)
	if count == 0 {
		return nil
	}
	res := C.rte_mempool_get_bulk(mp.ptr(), (*unsafe.Pointer)(ptr), C.uint(count))
	if res != 0 {
		return errors.New("mempool object allocation failed")
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
	C.rte_mempool_put_bulk(mp.ptr(), (*unsafe.Pointer)(ptr), C.uint(count))
}

// New creates a Mempool.
func New(cfg Config) (mp *Mempool, e error) {
	nameC := C.CString(eal.AllocObjectID("mempool.Mempool"))
	defer C.free(unsafe.Pointer(nameC))

	var flags C.unsigned
	if cfg.SingleProducer {
		flags |= C.RTE_MEMPOOL_F_SP_PUT
	}
	if cfg.SingleConsumer {
		flags |= C.RTE_MEMPOOL_F_SC_GET
	}

	capacity := ComputeOptimumCapacity(cfg.Capacity)
	cacheSize := ComputeCacheSize(capacity)
	mp = (*Mempool)(C.rte_mempool_create(nameC, C.uint(capacity), C.uint(cfg.ElementSize), C.uint(cacheSize),
		C.unsigned(cfg.PrivSize), nil, nil, nil, nil, C.int(cfg.Socket.ID()), flags))
	if mp == nil {
		return nil, eal.GetErrno()
	}
	return mp, nil
}

// FromPtr converts *C.struct_rte_mempool pointer to Mempool.
func FromPtr(ptr unsafe.Pointer) *Mempool {
	return (*Mempool)(ptr)
}
