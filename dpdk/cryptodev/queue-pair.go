package cryptodev

/*
#include "../../csrc/dpdk/cryptodev.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

// QueuePair represents a crypto device queue pair.
type QueuePair struct {
	c   C.CryptoQueuePair
	dev *CryptoDev
}

// Dev returns the CryptoDev.
func (qp *QueuePair) Dev() *CryptoDev {
	return qp.dev
}

// ID returns queue pair ID.
func (qp *QueuePair) ID() int {
	return int(qp.c.qp)
}

// CopyToC copies settings to *C.CryptoQueuePair struct.
func (qp *QueuePair) CopyToC(ptr unsafe.Pointer) {
	*(*C.CryptoQueuePair)(ptr) = qp.c
}

// PrepareSha256Digest prepares a SHA256 digest generation operation.
//  m[offset:offset+length] is the input to SHA256 digest function.
//  output must have 32 bytes in C memory.
func (qp *QueuePair) PrepareSha256(op *Op, m *pktmbuf.Packet, offset, length int, output unsafe.Pointer) {
	C.CryptoQueuePair_PrepareSha256(&qp.c, &op.op, (*C.struct_rte_mbuf)(m.Ptr()),
		C.uint32_t(offset), C.uint32_t(length), (*C.uint8_t)(output))
}

// EnqueueBurst submits a burst of crypto operations.
func (qp *QueuePair) EnqueueBurst(ops OpVector) int {
	return int(C.rte_cryptodev_enqueue_burst(qp.c.dev, qp.c.qp,
		cptr.FirstPtr[*C.struct_rte_crypto_op](ops), C.uint16_t(len(ops))))
}

// DequeueBurst retrieves a burst of completed crypto operations.
func (qp *QueuePair) DequeueBurst(ops OpVector) int {
	return int(C.rte_cryptodev_dequeue_burst(qp.c.dev, qp.c.qp,
		cptr.FirstPtr[*C.struct_rte_crypto_op](ops), C.uint16_t(len(ops))))
}
