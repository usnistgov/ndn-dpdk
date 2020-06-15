package cryptodev

/*
#include "../../csrc/dpdk/cryptodev.h"
*/
import "C"
import (
	"errors"
	"math"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/mempool"
)

// OpPoolConfig contains configuration for NewPool.
type OpPoolConfig struct {
	Capacity int
	PrivSize int
}

func (cfg *OpPoolConfig) applyDefaults() {
	if cfg.Capacity <= 0 {
		cfg.Capacity = 4095
	}
	if cfg.PrivSize < 0 || cfg.PrivSize > math.MaxUint16 {
		panic("PrivSize out of range")
	}
}

// OpPool represents a DPDK memory pool for crypto operations.
type OpPool struct {
	mempool.Mempool
}

// NewOpPool creates an OpPool.
func NewOpPool(name string, cfg OpPoolConfig, socket eal.NumaSocket) (mp *OpPool, e error) {
	cfg.applyDefaults()
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))

	mpC := C.rte_crypto_op_pool_create(nameC, C.RTE_CRYPTO_OP_TYPE_UNDEFINED, C.uint(cfg.Capacity),
		C.uint(mempool.ComputeCacheSize(cfg.Capacity)), C.uint16_t(cfg.PrivSize), C.int(socket.ID()))
	if mpC == nil {
		return nil, eal.GetErrno()
	}
	return OpPoolFromPtr(unsafe.Pointer(mpC)), nil
}

// OpPoolFromPtr converts *C.struct_rte_mempool pointer to OpPool.
func OpPoolFromPtr(ptr unsafe.Pointer) *OpPool {
	return (*OpPool)(ptr)
}

// Alloc allocates Op objects.
func (mp *OpPool) Alloc(opType OpType, count int) (vec OpVector, e error) {
	vec = make(OpVector, count)
	ptr, _ := cptr.ParseCptrArray(vec)
	res := C.rte_crypto_op_bulk_alloc((*C.struct_rte_mempool)(mp.GetPtr()), C.enum_rte_crypto_op_type(opType),
		(**C.struct_rte_crypto_op)(ptr), C.uint16_t(count))
	if res == 0 {
		return nil, errors.New("rte_crypto_op_bulk_alloc failed")
	}
	return vec, nil
}
