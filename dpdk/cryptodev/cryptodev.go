// Package cryptodev contains bindings of DPDK crypto device.
package cryptodev

/*
#include "../../csrc/dpdk/cryptodev.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"io"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/mempool"
)

// Config contains CryptoDev configuration.
type Config struct {
	MaxSessions       int
	NQueuePairs       int
	NQueueDescriptors int
}

func (cfg *Config) applyDefaults() {
	if cfg.NQueuePairs <= 0 {
		cfg.NQueuePairs = 1
	}
	if cfg.MaxSessions <= 0 {
		cfg.MaxSessions = 1024
	}
	if cfg.NQueueDescriptors <= 0 {
		cfg.NQueueDescriptors = 1024
	}
}

type device interface {
	io.Closer
	eal.WithNumaSocket
	Name() string
}

// CryptoDev represents a crypto device.
type CryptoDev struct {
	dev         device
	id          C.uint8_t
	sessionPool *mempool.Mempool
	queuePairs  []*QueuePair
}

// New initializes a crypto device.
func New(dev device, cfg Config) (cd *CryptoDev, e error) {
	cfg.applyDefaults()
	nameC := C.CString(dev.Name())
	defer C.free(unsafe.Pointer(nameC))
	socketC := C.int(dev.NumaSocket().ID())

	id := C.rte_cryptodev_get_dev_id(nameC)
	if id < 0 {
		return nil, fmt.Errorf("cryptodev %s not found", dev.Name())
	}

	cd = &CryptoDev{
		dev:        dev,
		id:         C.uint8_t(id),
		queuePairs: make([]*QueuePair, cfg.NQueuePairs),
	}

	mpNameC := C.CString(eal.AllocObjectID("cryptodev.SymSessionPool"))
	defer C.free(unsafe.Pointer(mpNameC))
	mpC := C.rte_cryptodev_sym_session_pool_create(mpNameC, C.uint32_t(cfg.MaxSessions*2),
		C.uint32_t(C.rte_cryptodev_sym_get_private_session_size(cd.id)), 0, 0, socketC)
	if mpC == nil {
		return nil, errors.New("rte_cryptodev_sym_session_pool_create error")
	}
	cd.sessionPool = mempool.FromPtr(unsafe.Pointer(mpC))

	var devConf C.struct_rte_cryptodev_config
	devConf.socket_id = socketC
	devConf.nb_queue_pairs = C.uint16_t(cfg.NQueuePairs)
	if res := C.rte_cryptodev_configure(cd.id, &devConf); res < 0 {
		return nil, fmt.Errorf("rte_cryptodev_configure error %d", res)
	}

	var qpConf C.struct_rte_cryptodev_qp_conf
	qpConf.nb_descriptors = C.uint32_t(cfg.NQueueDescriptors)
	qpConf.mp_session = mpC
	qpConf.mp_session_private = mpC
	for i := range cd.queuePairs {
		cd.queuePairs[i] = &QueuePair{cd, C.uint16_t(i)}
		if res := C.rte_cryptodev_queue_pair_setup(cd.id, C.uint16_t(i), &qpConf, socketC); res < 0 {
			return nil, fmt.Errorf("rte_cryptodev_queue_pair_setup(%d) error %d", i, res)
		}
	}

	if res := C.rte_cryptodev_start(cd.id); res < 0 {
		return nil, fmt.Errorf("rte_cryptodev_start error %d", res)
	}
	return cd, nil
}

// Close releases a crypto device.
func (cd *CryptoDev) Close() error {
	defer cd.sessionPool.Close()
	name := cd.Name()
	C.rte_cryptodev_stop(cd.id)
	if res := C.rte_cryptodev_close(cd.id); res < 0 {
		return fmt.Errorf("rte_cryptodev_close(%s) error %d", name, res)
	}
	return cd.dev.Close()
}

// ID returns crypto device ID.
func (cd *CryptoDev) ID() int {
	return int(cd.id)
}

// Name returns crypto device name.
func (cd *CryptoDev) Name() string {
	return cd.dev.Name()
}

// QueuePairs returns a list of queue pair.
func (cd *CryptoDev) QueuePairs() []*QueuePair {
	return cd.queuePairs
}
