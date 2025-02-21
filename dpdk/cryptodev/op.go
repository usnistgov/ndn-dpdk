package cryptodev

/*
#include "../../csrc/dpdk/cryptodev.h"

enum {
	c_offsetof_Op_status = offsetof(struct rte_crypto_op, status),
	c_sizeof_Op_sym_asym = RTE_MAX_T(sizeof(struct rte_crypto_sym_op), sizeof(struct rte_crypto_asym_op), size_t),
};
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// OpStatus indicates crypto operation status.
type OpStatus C.enum_rte_crypto_op_status

// OpStatus values.
const (
	OpStatusNew     = C.RTE_CRYPTO_OP_STATUS_NOT_PROCESSED
	OpStatusSuccess = C.RTE_CRYPTO_OP_STATUS_SUCCESS
)

// Op represents a crypto operation.
type Op struct {
	op  C.struct_rte_crypto_op
	buf [C.c_sizeof_Op_sym_asym]byte
}

// Ptr returns *C.struct_rte_crypto_op pointer.
func (op *Op) Ptr() unsafe.Pointer {
	return unsafe.Pointer(op)
}

// Status returns operation status.
func (op *Op) Status() OpStatus {
	return OpStatus(*(*C.uint8_t)(unsafe.Add(unsafe.Pointer(op), C.c_offsetof_Op_status)))
}

// Error returns an error if this operation has failed, otherwise returns nil.
func (op *Op) Error() error {
	s := op.Status()
	switch s {
	case OpStatusNew, OpStatusSuccess:
		return nil
	}
	return fmt.Errorf("CryptoOpStatus %d", s)
}

// OpVector represents a vector of crypto operations.
type OpVector []*Op
