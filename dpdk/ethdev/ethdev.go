// Package ethdev contains bindings of DPDK Ethernet device.
package ethdev

/*
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"
import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/core/pciaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/zap"
)

var logger = logging.New("ethdev")

// MaxEthDevs is maximum number of EthDevs.
const MaxEthDevs = C.RTE_MAX_ETHPORTS

// EthDev represents an Ethernet adapter.
type EthDev interface {
	eal.WithNumaSocket
	fmt.Stringer
	io.Closer

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

	// RxQueues returns RX queues of a running port.
	RxQueues() []RxQueue
	// TxQueues returns TX queues of a running port.
	TxQueues() []TxQueue

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
	var buf [C.RTE_ETH_NAME_MAX_LEN]C.char
	ifnameC := unsafe.SliceData(buf[:])
	res := C.rte_eth_dev_get_name_by_port(dev.cID(), ifnameC)
	if res != 0 {
		return ""
	}
	return C.GoString(ifnameC)
}

func (dev ethDev) NumaSocket() (socket eal.NumaSocket) {
	return eal.NumaSocketFromID(int(C.rte_eth_dev_socket_id(dev.cID())))
}

func (dev ethDev) DevInfo() (info DevInfo) {
	C.rte_eth_dev_info_get(dev.cID(), (*C.struct_rte_eth_dev_info)(unsafe.Pointer(&info.DevInfoC)))
	return
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
		zap.String("driver", info.Driver()),
		mtuField,
		zap.Int("rxq", len(cfg.RxQueues)),
		zap.Int("txq", len(cfg.TxQueues)),
		zap.Bool("promisc", cfg.Promisc),
	)
	bail := func(step string, res C.int) error {
		dev.stop(false)
		e := eal.MakeErrno(res)
		logEntry.Warn(step+" error", zap.Error(e))
		return fmt.Errorf("%s %w", step, e)
	}

	conf := C.struct_rte_eth_conf{}
	conf.rxmode.mtu = C.uint32_t(dev.MTU())
	conf.txmode.offloads = C.uint64_t(info.Tx_offload_capa & (txOffloadMultiSegs | txOffloadChecksum))

	if res := C.rte_eth_dev_configure(dev.cID(), C.uint16_t(len(cfg.RxQueues)), C.uint16_t(len(cfg.TxQueues)), &conf); res < 0 {
		return bail("rte_eth_dev_configure", res)
	}

	if cfg.MTU > 0 && cfg.MTU != dev.MTU() {
		if res := C.rte_eth_dev_set_mtu(dev.cID(), C.uint16_t(cfg.MTU)); res != 0 && !info.canIgnoreSetMTUError() {
			return bail("rte_eth_dev_set_mtu", res)
		}
	}

	for i, q := range cfg.RxQueues {
		capacity := info.Rx_desc_lim.adjustQueueCapacity(q.Capacity)
		if res := C.rte_eth_rx_queue_setup(dev.cID(), C.uint16_t(i), C.uint16_t(capacity), C.uint(q.Socket.ID()),
			(*C.struct_rte_eth_rxconf)(q.Conf), (*C.struct_rte_mempool)(q.RxPool.Ptr())); res != 0 {
			return bail(fmt.Sprintf("rte_eth_rx_queue_setup[%d]", i), res)
		}
	}

	for i, q := range cfg.TxQueues {
		capacity := info.Tx_desc_lim.adjustQueueCapacity(q.Capacity)
		if res := C.rte_eth_tx_queue_setup(dev.cID(), C.uint16_t(i), C.uint16_t(capacity), C.uint(q.Socket.ID()),
			(*C.struct_rte_eth_txconf)(q.Conf)); res != 0 {
			return bail(fmt.Sprintf("rte_eth_tx_queue_setup[%d]", i), res)
		}
	}

	if cfg.Promisc {
		if res := C.rte_eth_promiscuous_enable(dev.cID()); res != 0 && !info.canIgnorePromiscError() {
			return bail("rte_eth_promiscuous_enable", res)
		}
	} else {
		if res := C.rte_eth_promiscuous_disable(dev.cID()); res != 0 && !info.canIgnorePromiscError() {
			return bail("rte_eth_promiscuous_disable", res)
		}
	}

	if res := C.rte_eth_dev_start(dev.cID()); res != 0 {
		return bail("rte_eth_dev_start", res)
	}

	logEntry.Info("ethdev started")
	return nil
}

func (dev ethDev) stop(close bool) error {
	if C.rte_eth_dev_is_valid_port(dev.cID()) == 0 { // already detached
		return nil
	}

	logEntry := logger.With(
		zap.Int("id", dev.ID()),
		zap.String("name", dev.Name()),
	)
	bail := func(step string, res C.int) error {
		e := eal.MakeErrno(res)
		logEntry.Warn(step+" error", zap.Error(e))
		return fmt.Errorf("%s %w", step, e)
	}

	res := C.rte_eth_dev_stop(dev.cID())
	switch res {
	case 0, -C.ENOTSUP:
	default:
		return bail("rte_eth_dev_stop", res)
	}

	if close {
		if res := C.rte_eth_dev_close(dev.cID()); res != 0 {
			return bail("rte_eth_dev_close", res)
		}
		closeEmitter.Emit(dev.ID())
		logEntry.Info("ethdev stopped and closed")
		return nil
	}

	if res := C.rte_eth_dev_reset(dev.cID()); res != 0 {
		return bail("rte_eth_dev_reset", res)
	}
	logEntry.Info("ethdev stopped and reset")
	return nil
}

func (dev ethDev) Close() error {
	return dev.stop(true)
}

func (port ethDev) RxQueues() (list []RxQueue) {
	id, info := uint16(port.ID()), port.DevInfo()
	for queue := range info.Nb_rx_queues {
		list = append(list, RxQueue{id, queue})
	}
	return list
}

func (port ethDev) TxQueues() (list []TxQueue) {
	id, info := uint16(port.ID()), port.DevInfo()
	for queue := range info.Nb_tx_queues {
		list = append(list, TxQueue{id, queue})
	}
	return list
}

func (dev ethDev) Stats() (stats Stats) {
	C.rte_eth_stats_get(dev.cID(), (*C.struct_rte_eth_stats)(unsafe.Pointer(&stats)))
	stats.dev = dev
	return
}

func (dev ethDev) ResetStats() error {
	res := C.rte_eth_xstats_reset(dev.cID())
	return eal.MakeErrno(res)
}

// FromID converts port ID to EthDev.
func FromID(id int) EthDev {
	if id < 0 || id >= MaxEthDevs {
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

// FromHardwareAddr returns the first EthDev with specified MAC address.
func FromHardwareAddr(a net.HardwareAddr) EthDev {
	for p := C.rte_eth_find_next(0); p < C.RTE_MAX_ETHPORTS; p = C.rte_eth_find_next(p + 1) {
		dev := ethDev(p)
		if bytes.Equal(dev.HardwareAddr(), a) {
			return dev
		}
	}
	return nil
}

// FromPCI finds an EthDev from PCI address.
func FromPCI(addr pciaddr.PCIAddress) EthDev {
	return FromName(addr.String())
}

// ProbePCI requests to probe a PCI Ethernet adapter.
func ProbePCI(addr pciaddr.PCIAddress, args map[string]any) (EthDev, error) {
	if e := eal.ProbePCI(addr, args); e != nil {
		return nil, e
	}
	dev := FromPCI(addr)
	if dev == nil {
		return nil, errors.New("PCI probe did not create an Ethernet adapter")
	}
	return dev, nil
}
