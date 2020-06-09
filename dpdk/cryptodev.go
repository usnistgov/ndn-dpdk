package dpdk

/*
#include "cryptodev.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"strings"
	"unsafe"
)

// CryptoOpType represents a crypto operation type.
type CryptoOpType int

const (
	// CryptoOpSym indicates symmetric crypto operation type.
	CryptoOpSym CryptoOpType = C.RTE_CRYPTO_OP_TYPE_SYMMETRIC
)

func (t CryptoOpType) String() string {
	switch t {
	case CryptoOpSym:
		return "symmetric"
	}
	return fmt.Sprintf("%d", t)
}

// CryptoOp holds a pointer to a crypto operation structure.
type CryptoOp struct {
	c *C.struct_rte_crypto_op
	// DO NOT add other fields: *CryptoOp is passed to C code as rte_crypto_op**
}

// GetPtr returns the C pointer.
func (op CryptoOp) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(op.c)
}

// IsNew returns true if this operation has not been processed.
func (op CryptoOp) IsNew() bool {
	return C.CryptoOp_GetStatus(op.c) == C.RTE_CRYPTO_OP_STATUS_NOT_PROCESSED
}

// IsSuccess returns true if this operation has completed successfully.
func (op CryptoOp) IsSuccess() bool {
	return C.CryptoOp_GetStatus(op.c) == C.RTE_CRYPTO_OP_STATUS_SUCCESS
}

// Error returns an error if this operation has failed, otherwise returns nil.
func (op CryptoOp) Error() error {
	switch {
	case op.IsNew(), op.IsSuccess():
		return nil
	}
	return fmt.Errorf("CryptoOpStatus %d", C.CryptoOp_GetStatus(op.c))
}

// PrepareSha256Digest prepares a SHA256 digest generation operation.
// The input is from the given offset and length in the packet.
// The output must have 32 bytes in C memory.
func (op CryptoOp) PrepareSha256Digest(input Packet, offset, length int, output unsafe.Pointer) error {
	if offset < 0 || length < 0 || offset+length > input.Len() {
		return errors.New("offset+length exceeds packet boundary")
	}

	C.CryptoOp_PrepareSha256Digest(op.c, input.c, C.uint32_t(offset), C.uint32_t(length), (*C.uint8_t)(output))
	return nil
}

// Close discards this instance.
func (op CryptoOp) Close() error {
	mp := CryptoOpPool{Mempool{c: op.c.mempool}}
	mp.Free(unsafe.Pointer(op.c))
	return nil
}

// CryptoOpPool holds a pointer to a CryptoOp mempool.
type CryptoOpPool struct {
	Mempool
}

// NewCryptoOpPool creates a CryptoOpPool.
func NewCryptoOpPool(name string, capacity int, privSize int, socket NumaSocket) (mp CryptoOpPool, e error) {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))

	mp.c = C.rte_crypto_op_pool_create(nameC, C.RTE_CRYPTO_OP_TYPE_UNDEFINED,
		C.uint(capacity), C.uint(computeMempoolCacheSize(capacity)), C.uint16_t(privSize), C.int(socket.ID()))
	if mp.c == nil {
		return mp, GetErrno()
	}
	return mp, nil
}

// AllocBulk allocates CryptoOp objects.
func (mp CryptoOpPool) AllocBulk(opType CryptoOpType, ops []CryptoOp) error {
	ptr, count := ParseCptrArray(ops)
	res := C.rte_crypto_op_bulk_alloc(mp.c, C.enum_rte_crypto_op_type(opType),
		(**C.struct_rte_crypto_op)(ptr), C.uint16_t(count))
	if res == 0 {
		return errors.New("CryptoOp allocation failed")
	}
	return nil
}

// CryptoDev represents a crypto device.
type CryptoDev struct {
	devId       C.uint8_t
	sessionPool Mempool
	ownsVdev    bool
}

// NewCryptoDev initializes a crypto device.
func NewCryptoDev(name string, maxSessions, nQueuePairs int, socket NumaSocket) (cd CryptoDev, e error) {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	if devId := C.rte_cryptodev_get_dev_id(nameC); devId < 0 {
		return CryptoDev{}, fmt.Errorf("cryptodev %s not found", name)
	} else {
		cd.devId = C.uint8_t(devId)
	}

	mpNameC := C.CString(strings.TrimPrefix(name, "crypto_") + "_sess")
	defer C.free(unsafe.Pointer(mpNameC))
	if mpC := C.rte_cryptodev_sym_session_pool_create_(mpNameC, C.uint32_t(maxSessions*2),
		C.uint32_t(C.rte_cryptodev_sym_get_private_session_size(cd.devId)), 0, 0, C.int(socket.ID())); mpC == nil {
		return CryptoDev{}, errors.New("rte_cryptodev_sym_session_pool_create error")
	} else {
		cd.sessionPool.c = mpC
	}

	var devConf C.struct_rte_cryptodev_config
	devConf.socket_id = C.int(socket.ID())
	devConf.nb_queue_pairs = C.uint16_t(nQueuePairs)
	if res := C.rte_cryptodev_configure(cd.devId, &devConf); res < 0 {
		return CryptoDev{}, fmt.Errorf("rte_cryptodev_configure error %d", res)
	}

	var qpConf C.struct_rte_cryptodev_qp_conf
	qpConf.nb_descriptors = 2048
	qpConf.mp_session = cd.sessionPool.c
	qpConf.mp_session_private = cd.sessionPool.c
	for i := C.uint16_t(0); i < devConf.nb_queue_pairs; i++ {
		if res := C.rte_cryptodev_queue_pair_setup(cd.devId, i, &qpConf, C.int(socket.ID())); res < 0 {
			return CryptoDev{}, fmt.Errorf("rte_cryptodev_queue_pair_setup(%d) error %d", i, res)
		}
	}

	if res := C.rte_cryptodev_start(cd.devId); res < 0 {
		return CryptoDev{}, fmt.Errorf("rte_cryptodev_start error %d", res)
	}

	return cd, nil
}

// Close releases a crypto device.
func (cd CryptoDev) Close() error {
	defer cd.sessionPool.Close()
	name := cd.GetName()
	C.rte_cryptodev_stop(cd.devId)
	if res := C.rte_cryptodev_close(cd.devId); res < 0 {
		return fmt.Errorf("rte_cryptodev_close(%s) error %d", name, res)
	}
	if cd.ownsVdev {
		return DestroyVdev(name)
	}
	return nil
}

// ID returns crypto device ID.
func (cd CryptoDev) ID() int {
	return int(cd.devId)
}

// GetName returns crypto device name.
func (cd CryptoDev) GetName() string {
	return C.GoString(C.rte_cryptodev_name_get(cd.devId))
}

// GetQueuePair retrieves a crypto queue pair.
func (cd CryptoDev) GetQueuePair(i int) (qp CryptoQueuePair, ok bool) {
	qp.CryptoDev = cd
	qp.qpID = C.uint16_t(i)
	if qp.qpID >= C.rte_cryptodev_queue_pair_count(cd.devId) {
		return CryptoQueuePair{}, false
	}
	return qp, true
}

// CryptoQueuePair represents a crypto device queue pair.
type CryptoQueuePair struct {
	CryptoDev
	qpID C.uint16_t
}

// Copy to C.CryptoQueuePair .
func (qp CryptoQueuePair) CopyToC(ptr unsafe.Pointer) {
	c := (*C.CryptoQueuePair)(ptr)
	c.dev = qp.devId
	c.qp = qp.qpID
}

// Submit a burst of crypto operations.
func (qp CryptoQueuePair) EnqueueBurst(ops []CryptoOp) int {
	ptr, count := ParseCptrArray(ops)
	if count == 0 {
		return 0
	}
	res := C.rte_cryptodev_enqueue_burst(qp.devId, qp.qpID,
		(**C.struct_rte_crypto_op)(ptr), C.uint16_t(count))
	return int(res)
}

// Retrieve a burst of completed crypto operations.
func (qp CryptoQueuePair) DequeueBurst(ops []CryptoOp) int {
	ptr, count := ParseCptrArray(ops)
	if count == 0 {
		return 0
	}
	res := C.rte_cryptodev_dequeue_burst(qp.devId, qp.qpID,
		(**C.struct_rte_crypto_op)(ptr), C.uint16_t(count))
	return int(res)
}

// CryptoDevDriverPref is a priority list of CryptoDev drivers.
type CryptoDevDriverPref []string

var (
	// CryptoDrvSingleSeg lists CryptoDev drivers capable of computing SHA256 on single-segment mbufs.
	CryptoDrvSingleSeg = CryptoDevDriverPref{"aesni_mb", "openssl"}

	// CryptoDrvMultiSeg lists CryptoDev drivers capable of computing SHA256 on multi-segment mbufs.
	CryptoDrvMultiSeg = CryptoDevDriverPref{"openssl"}
)

// Create constructs a CryptoDev from a list of drivers.
func (drvs CryptoDevDriverPref) Create(id string, nQueuePairs int, socket NumaSocket) (cd CryptoDev, e error) {
	args := fmt.Sprintf("max_nb_queue_pairs=%d", nQueuePairs)
	if !socket.IsAny() {
		args += fmt.Sprintf(",socket_id=%d", socket)
	}

	var name string
	var drvErrors []string
	for _, drv := range drvs {
		name = fmt.Sprintf("crypto_%s_%s", drv, id)
		if e := CreateVdev(name, args); e != nil {
			drvErrors = append(drvErrors, fmt.Sprintf("%s: %s", drv, e))
			name = ""
		} else {
			break
		}
	}
	if name == "" {
		return CryptoDev{}, fmt.Errorf("virtual cryptodev unavailable: %s", strings.Join(drvErrors, "; "))
	}

	if cd, e = NewCryptoDev(name, 1024, nQueuePairs, socket); e != nil {
		return CryptoDev{}, e
	}
	cd.ownsVdev = true
	return cd, nil
}
