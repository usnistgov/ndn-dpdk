package cryptodev

/*
#include "cryptodev.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/core/cptr"
)

// QueuePair represents a crypto device queue pair.
type QueuePair struct {
	*CryptoDev
	qpID C.uint16_t
}

// CopyToC copies settings to to *C.CryptoQueuePair struct.
func (qp *QueuePair) CopyToC(ptr unsafe.Pointer) {
	c := (*C.CryptoQueuePair)(ptr)
	c.dev = qp.devID
	c.qp = qp.qpID
}

// EnqueueBurst submits a burst of crypto operations.
func (qp *QueuePair) EnqueueBurst(ops OpVector) int {
	ptr, count := cptr.ParseCptrArray(ops)
	if count == 0 {
		return 0
	}
	res := C.rte_cryptodev_enqueue_burst(qp.devID, qp.qpID,
		(**C.struct_rte_crypto_op)(ptr), C.uint16_t(count))
	return int(res)
}

// DequeueBurst retrieves a burst of completed crypto operations.
func (qp *QueuePair) DequeueBurst(ops OpVector) int {
	ptr, count := cptr.ParseCptrArray(ops)
	if count == 0 {
		return 0
	}
	res := C.rte_cryptodev_dequeue_burst(qp.devID, qp.qpID,
		(**C.struct_rte_crypto_op)(ptr), C.uint16_t(count))
	return int(res)
}
