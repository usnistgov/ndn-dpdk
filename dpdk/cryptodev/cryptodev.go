package cryptodev

/*
#include "../../csrc/dpdk/cryptodev.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"strings"
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

// CryptoDev represents a crypto device.
type CryptoDev struct {
	devID       C.uint8_t
	sessionPool *mempool.Mempool
	ownsVdev    bool
	queuePairs  []*QueuePair
}

// New initializes a crypto device.
func New(name string, cfg Config, socket eal.NumaSocket) (cd *CryptoDev, e error) {
	cfg.applyDefaults()
	cd = new(CryptoDev)

	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	devID := C.rte_cryptodev_get_dev_id(nameC)
	if devID < 0 {
		return nil, fmt.Errorf("cryptodev %s not found", name)
	}
	cd.devID = C.uint8_t(devID)

	mpNameC := C.CString(eal.AllocObjectID("cryptodev.SymSessionPool"))
	defer C.free(unsafe.Pointer(mpNameC))
	mpC := C.rte_cryptodev_sym_session_pool_create_(mpNameC, C.uint32_t(cfg.MaxSessions*2),
		C.uint32_t(C.rte_cryptodev_sym_get_private_session_size(cd.devID)), 0, 0, C.int(socket.ID()))
	if mpC == nil {
		return nil, errors.New("rte_cryptodev_sym_session_pool_create error")
	}
	cd.sessionPool = mempool.FromPtr(unsafe.Pointer(mpC))

	var devConf C.struct_rte_cryptodev_config
	devConf.socket_id = C.int(socket.ID())
	devConf.nb_queue_pairs = C.uint16_t(cfg.NQueuePairs)
	if res := C.rte_cryptodev_configure(cd.devID, &devConf); res < 0 {
		return nil, fmt.Errorf("rte_cryptodev_configure error %d", res)
	}

	var qpConf C.struct_rte_cryptodev_qp_conf
	qpConf.nb_descriptors = C.uint32_t(cfg.NQueueDescriptors)
	qpConf.mp_session = mpC
	qpConf.mp_session_private = mpC
	for i := C.uint16_t(0); i < devConf.nb_queue_pairs; i++ {
		if res := C.rte_cryptodev_queue_pair_setup(cd.devID, i, &qpConf, C.int(socket.ID())); res < 0 {
			return nil, fmt.Errorf("rte_cryptodev_queue_pair_setup(%d) error %d", i, res)
		}
	}

	if res := C.rte_cryptodev_start(cd.devID); res < 0 {
		return nil, fmt.Errorf("rte_cryptodev_start error %d", res)
	}

	cd.queuePairs = make([]*QueuePair, cfg.NQueuePairs)
	for i := range cd.queuePairs {
		cd.queuePairs[i] = &QueuePair{cd, C.uint16_t(i)}
	}

	return cd, nil
}

// Close releases a crypto device.
func (cd *CryptoDev) Close() error {
	defer cd.sessionPool.Close()
	name := cd.Name()
	C.rte_cryptodev_stop(cd.devID)
	if res := C.rte_cryptodev_close(cd.devID); res < 0 {
		return fmt.Errorf("rte_cryptodev_close(%s) error %d", name, res)
	}
	if cd.ownsVdev {
		return eal.DestroyVdev(name)
	}
	return nil
}

// ID returns crypto device ID.
func (cd *CryptoDev) ID() int {
	return int(cd.devID)
}

// Name returns crypto device name.
func (cd *CryptoDev) Name() string {
	return C.GoString(C.rte_cryptodev_name_get(cd.devID))
}

// QueuePair returns i-th queue pair.
func (cd *CryptoDev) QueuePair(i int) *QueuePair {
	if i < 0 || i >= len(cd.queuePairs) {
		return nil
	}
	return cd.queuePairs[i]
}

// DriverPref is a priority list of CryptoDev drivers.
type DriverPref []string

var (
	// SingleSegDrv lists CryptoDev drivers capable of computing SHA256 on single-segment mbufs.
	SingleSegDrv = DriverPref{"aesni_mb", "openssl"}

	// MultiSegDrv lists CryptoDev drivers capable of computing SHA256 on multi-segment mbufs.
	MultiSegDrv = DriverPref{"openssl"}
)

// Create constructs a CryptoDev from a list of drivers.
func (drvs DriverPref) Create(cfg Config, socket eal.NumaSocket) (cd *CryptoDev, e error) {
	cfg.applyDefaults()
	args := fmt.Sprintf("max_nb_queue_pairs=%d", cfg.NQueuePairs)
	if !socket.IsAny() {
		args += fmt.Sprintf(",socket_id=%d", socket.ID())
	}

	var name string
	var drvErrors []string
	for _, drv := range drvs {
		name = fmt.Sprintf("crypto_%s_%s", drv, eal.AllocObjectID("cryptodev.Driver["+drv+"]"))
		if e := eal.CreateVdev(name, args); e != nil {
			drvErrors = append(drvErrors, fmt.Sprintf("%s: %s", drv, e))
			name = ""
		} else {
			break
		}
	}
	if name == "" {
		return nil, fmt.Errorf("virtual cryptodev unavailable: %s", strings.Join(drvErrors, "; "))
	}

	if cd, e = New(name, cfg, socket); e != nil {
		return nil, e
	}
	cd.ownsVdev = true
	return cd, nil
}
