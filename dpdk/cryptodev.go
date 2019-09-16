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

// Crypto operation type.
type CryptoOpType int

const (
	CRYPTO_OP_SYM CryptoOpType = C.RTE_CRYPTO_OP_TYPE_SYMMETRIC
)

func (t CryptoOpType) String() string {
	switch t {
	case CRYPTO_OP_SYM:
		return "symmetric"
	}
	return fmt.Sprintf("%d", t)
}

// Crypto operation status.
type CryptoOpStatus int

const (
	CRYPTO_OP_SUCCESS  CryptoOpStatus = C.RTE_CRYPTO_OP_STATUS_SUCCESS
	CRYPTO_OP_NEW      CryptoOpStatus = C.RTE_CRYPTO_OP_STATUS_NOT_PROCESSED
	CRYPTO_OP_AUTHFAIL CryptoOpStatus = C.RTE_CRYPTO_OP_STATUS_AUTH_FAILED
	CRYPTO_OP_BADARG   CryptoOpStatus = C.RTE_CRYPTO_OP_STATUS_INVALID_ARGS
	CRYPTO_OP_ERROR    CryptoOpStatus = C.RTE_CRYPTO_OP_STATUS_ERROR
)

func (s CryptoOpStatus) String() string {
	switch s {
	case CRYPTO_OP_SUCCESS:
		return "success"
	case CRYPTO_OP_NEW:
		return "new"
	case CRYPTO_OP_AUTHFAIL:
		return "authfail"
	case CRYPTO_OP_BADARG:
		return "badarg"
	case CRYPTO_OP_ERROR:
		return "error"
	}
	return fmt.Sprintf("%d", s)
}

func (s CryptoOpStatus) Error() string {
	if s == CRYPTO_OP_SUCCESS {
		panic("not an error")
	}
	return fmt.Sprintf("CryptoOp-%s", s)
}

// Crypto operation.
type CryptoOp struct {
	c *C.struct_rte_crypto_op
	// DO NOT add other fields: *CryptoOp is passed to C code as rte_crypto_op**
}

func (op CryptoOp) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(op.c)
}

func (op CryptoOp) GetStatus() CryptoOpStatus {
	return CryptoOpStatus(C.CryptoOp_GetStatus(op.c))
}

// Setup SHA256 digest generation operation.
// m: input packet.
// offset, length: range of input packet.
// output: digest output, 32 bytes in C memory.
func (op CryptoOp) PrepareSha256Digest(m Packet, offset, length int, output unsafe.Pointer) error {
	if offset < 0 || length < 0 || offset+length > m.Len() {
		return errors.New("offset+length exceeds packet boundary")
	}

	C.CryptoOp_PrepareSha256Digest(op.c, m.c, C.uint32_t(offset), C.uint32_t(length), (*C.uint8_t)(output))
	return nil
}

func (op CryptoOp) Close() error {
	mp := CryptoOpPool{Mempool{c: op.c.mempool}}
	mp.Free(unsafe.Pointer(op.c))
	return nil
}

// Mempool for CryptoOp.
type CryptoOpPool struct {
	Mempool
}

func NewCryptoOpPool(name string, capacity int, cacheSize int, privSize int, socket NumaSocket) (mp CryptoOpPool, e error) {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))

	mp.c = C.rte_crypto_op_pool_create(nameC, C.RTE_CRYPTO_OP_TYPE_UNDEFINED,
		C.uint(capacity), C.uint(cacheSize), C.uint16_t(privSize), C.int(socket))
	if mp.c == nil {
		return mp, GetErrno()
	}
	return mp, nil
}

func (mp CryptoOpPool) AllocBulk(opType CryptoOpType, ops []CryptoOp) error {
	ptr, count := ParseCptrArray(ops)
	res := C.rte_crypto_op_bulk_alloc(mp.c, C.enum_rte_crypto_op_type(opType),
		(**C.struct_rte_crypto_op)(ptr), C.uint16_t(count))
	if res == 0 {
		return errors.New("CryptoOp allocation failed")
	}
	return nil
}

// Crypto device.
type CryptoDev struct {
	devId       C.uint8_t
	sessionPool Mempool
	ownsVdev    bool
}

// Initialize a crypto device.
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
		C.uint32_t(C.rte_cryptodev_sym_get_private_session_size(cd.devId)), 0, 0, C.int(socket)); mpC == nil {
		return CryptoDev{}, errors.New("rte_cryptodev_sym_session_pool_create error")
	} else {
		cd.sessionPool.c = mpC
	}

	var devConf C.struct_rte_cryptodev_config
	devConf.socket_id = C.int(socket)
	devConf.nb_queue_pairs = C.uint16_t(nQueuePairs)
	if res := C.rte_cryptodev_configure(cd.devId, &devConf); res < 0 {
		return CryptoDev{}, fmt.Errorf("rte_cryptodev_configure error %d", res)
	}

	var qpConf C.struct_rte_cryptodev_qp_conf
	qpConf.nb_descriptors = 2048
	qpConf.mp_session = cd.sessionPool.c
	qpConf.mp_session_private = cd.sessionPool.c
	for i := C.uint16_t(0); i < devConf.nb_queue_pairs; i++ {
		if res := C.rte_cryptodev_queue_pair_setup(cd.devId, i, &qpConf, C.int(socket)); res < 0 {
			return CryptoDev{}, fmt.Errorf("rte_cryptodev_queue_pair_setup(%d) error %d", i, res)
		}
	}

	if res := C.rte_cryptodev_start(cd.devId); res < 0 {
		return CryptoDev{}, fmt.Errorf("rte_cryptodev_start error %d", res)
	}

	return cd, nil
}

// Create an OpenSSL virtual crypto device.
func NewOpensslCryptoDev(id string, nQueuePairs int, socket NumaSocket) (cd CryptoDev, e error) {
	name := fmt.Sprintf("crypto_openssl_%s", id)
	var args string
	if socket != NUMA_SOCKET_ANY {
		args = fmt.Sprintf("socket_id=%d", socket)
	}
	if e = CreateVdev(name, args); e != nil {
		return CryptoDev{}, e
	}
	if cd, e = NewCryptoDev(name, 1024, nQueuePairs, socket); e != nil {
		return CryptoDev{}, e
	}
	cd.ownsVdev = true
	return cd, nil
}

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

func (cd CryptoDev) GetId() int {
	return int(cd.devId)
}

func (cd CryptoDev) GetName() string {
	return C.GoString(C.rte_cryptodev_name_get(cd.devId))
}

func (cd CryptoDev) GetQueuePair(i int) (qp CryptoQueuePair, ok bool) {
	qp.CryptoDev = cd
	qp.qpId = C.uint16_t(i)
	if qp.qpId >= C.rte_cryptodev_queue_pair_count(cd.devId) {
		return CryptoQueuePair{}, false
	}
	return qp, true
}

// Crypto device queue pair.
type CryptoQueuePair struct {
	CryptoDev
	qpId C.uint16_t
}

// Submit a burst of crypto operations.
func (qp CryptoQueuePair) EnqueueBurst(ops []CryptoOp) int {
	ptr, count := ParseCptrArray(ops)
	if count == 0 {
		return 0
	}
	res := C.rte_cryptodev_enqueue_burst(qp.devId, qp.qpId,
		(**C.struct_rte_crypto_op)(ptr), C.uint16_t(count))
	return int(res)
}

// Retrieve a burst of completed crypto operations.
func (qp CryptoQueuePair) DequeueBurst(ops []CryptoOp) int {
	ptr, count := ParseCptrArray(ops)
	if count == 0 {
		return 0
	}
	res := C.rte_cryptodev_dequeue_burst(qp.devId, qp.qpId,
		(**C.struct_rte_crypto_op)(ptr), C.uint16_t(count))
	return int(res)
}
