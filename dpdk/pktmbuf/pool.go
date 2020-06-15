package pktmbuf

/*
#include "../../csrc/dpdk/mbuf.h"
*/
import "C"
import (
	"errors"
	"math"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/mempool"
)

// PoolConfig contains configuration for NewPool.
type PoolConfig struct {
	Capacity int
	PrivSize int
	Dataroom int
}

func (cfg *PoolConfig) applyDefaults() {
	if cfg.Capacity <= 0 {
		cfg.Capacity = 4095
	}
	if cfg.PrivSize < 0 || cfg.PrivSize > math.MaxUint16 {
		panic("PrivSize out of range")
	}
	if cfg.Dataroom < 0 || cfg.Dataroom > math.MaxUint16 {
		panic("Dataroom out of range")
	}
}

// Pool represents a DPDK memory pool for packet buffers.
type Pool struct {
	mempool.Mempool
}

// NewPool creates a Pool.
func NewPool(name string, cfg PoolConfig, socket eal.NumaSocket) (mp *Pool, e error) {
	cfg.applyDefaults()
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))

	mpC := C.rte_pktmbuf_pool_create(nameC, C.uint(cfg.Capacity), C.uint(mempool.ComputeCacheSize(cfg.Capacity)),
		C.uint16_t(cfg.PrivSize), C.uint16_t(cfg.Dataroom), C.int(socket.ID()))
	if mpC == nil {
		return nil, eal.GetErrno()
	}
	return PoolFromPtr(unsafe.Pointer(mpC)), nil
}

// PoolFromPtr converts *C.struct_rte_mempool pointer to Pool.
func PoolFromPtr(ptr unsafe.Pointer) *Pool {
	return (*Pool)(ptr)
}

func (mp *Pool) getPtr() *C.struct_rte_mempool {
	return (*C.struct_rte_mempool)(mp.GetPtr())
}

// GetDataroom returns dataroom setting.
func (mp *Pool) GetDataroom() int {
	return int(C.rte_pktmbuf_data_room_size(mp.getPtr()))
}

// Alloc allocates a vector of mbufs.
func (mp *Pool) Alloc(count int) (vec Vector, e error) {
	vec = make(Vector, count)
	res := C.rte_pktmbuf_alloc_bulk(mp.getPtr(), vec.getPtr(), C.uint(count))
	if res != 0 {
		return Vector{}, errors.New("rte_pktmbuf_alloc_bulk failed")
	}
	return vec, nil
}

// MustAlloc allocates a vector of mbufs, and panics upon error
func (mp *Pool) MustAlloc(count int) Vector {
	vec, e := mp.Alloc(count)
	if e != nil {
		panic(e)
	}
	return vec
}
