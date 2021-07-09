package cryptodev

/*
#include "../../csrc/dpdk/cryptodev.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

// QueuePair represents a crypto device queue pair.
type QueuePair struct {
	dev *CryptoDev
	id  C.uint16_t
}

// Dev returns the CryptoDev.
func (qp *QueuePair) Dev() *CryptoDev {
	return qp.dev
}

// ID returns queue pair ID.
func (qp *QueuePair) ID() int {
	return int(qp.id)
}

// CopyToC copies settings to *C.CryptoQueuePair struct.
func (qp *QueuePair) CopyToC(ptr unsafe.Pointer) {
	c := (*C.CryptoQueuePair)(ptr)
	c.dev = qp.dev.id
	c.qp = qp.id
}

// EnqueueBurst submits a burst of crypto operations.
func (qp *QueuePair) EnqueueBurst(ops OpVector) int {
	ptr, count := cptr.ParseCptrArray(ops)
	if count == 0 {
		return 0
	}
	res := C.rte_cryptodev_enqueue_burst(qp.dev.id, qp.id, (**C.struct_rte_crypto_op)(ptr), C.uint16_t(count))
	return int(res)
}

// DequeueBurst retrieves a burst of completed crypto operations.
func (qp *QueuePair) DequeueBurst(ops OpVector) int {
	ptr, count := cptr.ParseCptrArray(ops)
	if count == 0 {
		return 0
	}
	res := C.rte_cryptodev_dequeue_burst(qp.dev.id, qp.id, (**C.struct_rte_crypto_op)(ptr), C.uint16_t(count))
	return int(res)
}
