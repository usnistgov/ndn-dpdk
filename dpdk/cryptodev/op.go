package cryptodev

/*
#include "../../csrc/dpdk/cryptodev.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"strconv"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

// OpType represents a crypto operation type.
type OpType int

const (
	// OpSymmetric indicates symmetric crypto operation type.
	OpSymmetric OpType = C.RTE_CRYPTO_OP_TYPE_SYMMETRIC
)

func (t OpType) String() string {
	switch t {
	case OpSymmetric:
		return "symmetric"
	}
	return strconv.Itoa(int(t))
}

// Op holds a pointer to a crypto operation structure.
type Op C.struct_rte_crypto_op

// GetPtr returns *C.struct_rte_crypto_op pointer.
func (op *Op) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(op)
}

func (op *Op) getPtr() *C.struct_rte_crypto_op {
	return (*C.struct_rte_crypto_op)(op)
}

// IsNew returns true if this operation has not been processed.
func (op *Op) IsNew() bool {
	return C.CryptoOp_GetStatus(op.getPtr()) == C.RTE_CRYPTO_OP_STATUS_NOT_PROCESSED
}

// IsSuccess returns true if this operation has completed successfully.
func (op *Op) IsSuccess() bool {
	return C.CryptoOp_GetStatus(op.getPtr()) == C.RTE_CRYPTO_OP_STATUS_SUCCESS
}

// Error returns an error if this operation has failed, otherwise returns nil.
func (op *Op) Error() error {
	switch {
	case op.IsNew(), op.IsSuccess():
		return nil
	}
	return fmt.Errorf("CryptoOpStatus %d", C.CryptoOp_GetStatus(op.getPtr()))
}

// PrepareSha256Digest prepares a SHA256 digest generation operation.
// The input is from the given offset and length in the packet.
// The output must have 32 bytes in C memory.
func (op *Op) PrepareSha256Digest(input *pktmbuf.Packet, offset, length int, output unsafe.Pointer) error {
	if offset < 0 || length < 0 || offset+length > input.Len() {
		return errors.New("offset+length exceeds packet boundary")
	}

	C.CryptoOp_PrepareSha256Digest(op.getPtr(), (*C.struct_rte_mbuf)(input.GetPtr()),
		C.uint32_t(offset), C.uint32_t(length), (*C.uint8_t)(output))
	return nil
}

// Close discards this instance.
func (op *Op) Close() error {
	C.rte_crypto_op_free(op.getPtr())
	return nil
}

// OpVector represents a vector of crypto operations.
type OpVector []*Op
