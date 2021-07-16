// Package ethdev contains bindings of DPDK Ethernet device.
package ethdev

/*
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"
import (
	"fmt"
	"net"
	"strconv"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/zap"
)

var logger = logging.New("ethdev")

// EthDev represents an Ethernet adapter.
type EthDev interface {
	fmt.Stringer
	eal.WithNumaSocket

	// ID returns DPDK ethdev ID.
	ID() int
	// ZapField returns a zap.Field for logging.
	ZapField(key string) zap.Field
	// Name returns port name.
	Name() string

	// DevInfo returns information about the hardware device.
	DevInfo() DevInfo
	// HardwareAddr returns the device MAC address.
	HardwareAddr() net.HardwareAddr
	// MTU returns MTU of this EthDev.
	MTU() int
	// IsDown determines whether this device is down.
	IsDown() bool

	// Start configures and starts this device.
	Start(cfg Config) error
	// Stop stops this device.
	Stop(mode StopMode) error

	// RxQueues returns RX queues of a running port.
	RxQueues() (list []RxQueue)
	// TxQueues returns TX queues of a running port.
	TxQueues() (list []TxQueue)

	// Stats retrieves hardware statistics.
	Stats() Stats
	// ResetStats clears hardware statistics.
	ResetStats() error
}

type ethDev int

func (dev ethDev) ID() int {
	return int(dev)
}

func (dev ethDev) cID() C.uint16_t {
	return C.uint16_t(dev)
}

func (dev ethDev) ZapField(key string) zap.Field {
	return zap.Int(key, dev.ID())
}

func (dev ethDev) String() string {
	return strconv.Itoa(dev.ID())
}

func (dev ethDev) Name() string {
	var ifname [C.RTE_ETH_NAME_MAX_LEN]C.char
	res := C.rte_eth_dev_get_name_by_port(dev.cID(), &ifname[0])
	if res != 0 {
		return ""
	}
	return C.GoString(&ifname[0])
}

func (dev ethDev) NumaSocket() (socket eal.NumaSocket) {
	return eal.NumaSocketFromID(int(C.rte_eth_dev_socket_id(dev.cID())))
}

func (dev ethDev) DevInfo() (info DevInfo) {
	C.rte_eth_dev_info_get(dev.cID(), (*C.struct_rte_eth_dev_info)(unsafe.Pointer(&info)))
	return info
}

func (dev ethDev) HardwareAddr() (a net.HardwareAddr) {
	var macC C.struct_rte_ether_addr
	C.rte_eth_macaddr_get(dev.cID(), &macC)
	return net.HardwareAddr(C.GoBytes(unsafe.Pointer(&macC.addr_bytes[0]), C.RTE_ETHER_ADDR_LEN))
}

func (dev ethDev) MTU() int {
	var mtu C.uint16_t
	C.rte_eth_dev_get_mtu(dev.cID(), &mtu)
	return int(mtu)
}

func (dev ethDev) IsDown() bool {
	return bool(C.EthDev_IsDown(dev.cID()))
}

func (dev ethDev) Start(cfg Config) error {
	info := dev.DevInfo()
	if info.Max_rx_queues > 0 && len(cfg.RxQueues) > int(info.Max_rx_queues) {
		return fmt.Errorf("cannot add more than %d RX queues", info.Max_rx_queues)
	}
	if info.Max_tx_queues > 0 && len(cfg.TxQueues) > int(info.Max_tx_queues) {
		return fmt.Errorf("cannot add more than %d TX queues", info.Max_tx_queues)
	}

	mtuField := zap.Int("mtu", cfg.MTU)
	if cfg.MTU == 0 {
		mtuField = zap.String("mtu", "unchanged")
	}
	logEntry := logger.With(
		zap.Int("id", dev.ID()),
		zap.String("name", dev.Name()),
		mtuField,
		zap.Int("rxq", len(cfg.RxQueues)),
		zap.Int("txq", len(cfg.TxQueues)),
		zap.Bool("promisc", cfg.Promisc),
	)
	bail := func(e error) error {
		dev.Stop(StopReset)
		logEntry.Warn("Start error", zap.Error(e))
		return e
	}

	if cfg.MTU > 0 {
		if res := C.rte_eth_dev_set_mtu(dev.cID(), C.uint16_t(cfg.MTU)); res != 0 {
			return bail(fmt.Errorf("rte_eth_dev_set_mtu(%d,%d) %w", dev, cfg.MTU, eal.Errno(-res)))
		}
	}

	conf := (*C.struct_rte_eth_conf)(cfg.Conf)
	if conf == nil {
		conf = &C.struct_rte_eth_conf{}
		conf.rxmode.max_rx_pkt_len = C.uint32_t(dev.MTU())
		conf.txmode.offloads = C.uint64_t(info.Tx_offload_capa & (txOffloadMultiSegs | txOffloadChecksum))
	}

	res := C.rte_eth_dev_configure(dev.cID(), C.uint16_t(len(cfg.RxQueues)), C.uint16_t(len(cfg.TxQueues)), conf)
	if res < 0 {
		return bail(fmt.Errorf("rte_eth_dev_configure(%d) %w", dev, eal.Errno(-res)))
	}

	for i, qcfg := range cfg.RxQueues {
		capacity := info.Rx_desc_lim.adjustQueueCapacity(qcfg.Capacity)
		res = C.rte_eth_rx_queue_setup(dev.cID(), C.uint16_t(i), C.uint16_t(capacity),
			C.uint(qcfg.Socket.ID()), (*C.struct_rte_eth_rxconf)(qcfg.Conf), (*C.struct_rte_mempool)(qcfg.RxPool.Ptr()))
		if res != 0 {
			return bail(fmt.Errorf("rte_eth_rx_queue_setup(%d,%d) %w", dev, i, eal.Errno(-res)))
		}
	}

	for i, qcfg := range cfg.TxQueues {
		capacity := info.Tx_desc_lim.adjustQueueCapacity(qcfg.Capacity)
		res = C.rte_eth_tx_queue_setup(dev.cID(), C.uint16_t(i), C.uint16_t(capacity),
			C.uint(qcfg.Socket.ID()), (*C.struct_rte_eth_txconf)(qcfg.Conf))
		if res != 0 {
			return bail(fmt.Errorf("rte_eth_tx_queue_setup(%d,%d) %w", dev, i, eal.Errno(-res)))
		}
	}

	if cfg.Promisc {
		C.rte_eth_promiscuous_enable(dev.cID())
	} else {
		C.rte_eth_promiscuous_disable(dev.cID())
	}

	res = C.rte_eth_dev_start(dev.cID())
	if res != 0 {
		return bail(fmt.Errorf("rte_eth_dev_start(%d) %w", dev, eal.Errno(-res)))
	}

	logEntry.Info("ethdev started")
	return nil
}

func (dev ethDev) Stop(mode StopMode) error {
	logEntry := logger.With(
		zap.Int("id", dev.ID()),
		zap.String("name", dev.Name()),
	)

	res := C.rte_eth_dev_stop(dev.cID())
	switch res {
	case 0, -C.ENOTSUP:
	case -C.ENODEV: // already detached
		return nil
	default:
		e := eal.Errno(-res)
		logEntry.Warn("rte_eth_dev_stop error", zap.Error(e))
		return e
	}

	switch mode {
	case StopDetach:
		if res := C.rte_eth_dev_close(dev.cID()); res != 0 {
			e := eal.Errno(-res)
			logEntry.Warn("rte_eth_dev_close error", zap.Error(e))
			return e
		}
		detachEmitter.Emit(dev.ID())
		logEntry.Info("stopped and detached")
		return nil
	case StopReset:
		if res := C.rte_eth_dev_reset(dev.cID()); res != 0 {
			e := eal.Errno(-res)
			logEntry.Warn("rte_eth_dev_reset error", zap.Error(e))
			return e
		}
		logEntry.Info("ethdev stopped and reset")
		return nil
	}
	panic(mode)
}

func (dev ethDev) Stats() (stats Stats) {
	C.rte_eth_stats_get(dev.cID(), (*C.struct_rte_eth_stats)(unsafe.Pointer(&stats)))
	return
}

func (dev ethDev) ResetStats() error {
	res := C.rte_eth_stats_reset(dev.cID())
	return eal.MakeErrno(res)
}

// FromID converts port ID to EthDev.
func FromID(id int) EthDev {
	if id < 0 || id >= C.RTE_MAX_ETHPORTS {
		return nil
	}

	if p := C.rte_eth_find_next(C.uint16_t(id)); p != C.uint16_t(id) {
		return nil
	}

	return ethDev(id)
}

// List returns a list of Ethernet adapters.
func List() (list []EthDev) {
	for p := C.rte_eth_find_next(0); p < C.RTE_MAX_ETHPORTS; p = C.rte_eth_find_next(p + 1) {
		list = append(list, ethDev(p))
	}
	return list
}

// FromName retrieves Ethernet adapter by name.
func FromName(name string) EthDev {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))

	var port C.uint16_t
	if res := C.rte_eth_dev_get_port_by_name(nameC, &port); res != 0 {
		return nil
	}

	return ethDev(port)
}
