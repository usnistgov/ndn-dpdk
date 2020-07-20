package ethdev

/*
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"
import (
	"fmt"
	"strconv"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
)

// EthDev represents an Ethernet adapter.
type EthDev struct {
	v int // ethdev ID + 1
}

// FromID converts port ID to EthDev.
func FromID(id int) EthDev {
	if id < 0 || id >= C.RTE_MAX_ETHPORTS {
		return EthDev{}
	}
	return EthDev{id + 1}
}

// List returns a list of Ethernet adapters.
func List() (list []EthDev) {
	for p := C.rte_eth_find_next(0); p < C.RTE_MAX_ETHPORTS; p = C.rte_eth_find_next(p + 1) {
		list = append(list, FromID(int(p)))
	}
	return list
}

// Find locates an EthDev by name.
func Find(name string) EthDev {
	for p := C.rte_eth_find_next(0); p < C.RTE_MAX_ETHPORTS; p = C.rte_eth_find_next(p + 1) {
		port := FromID(int(p))
		if port.Name() == name {
			return port
		}
	}
	return EthDev{}
}

// ID returns EthDev ID.
func (port EthDev) ID() int {
	return port.v - 1
}

// Valid returns true if this is a valid Ethernet port.
func (port EthDev) Valid() bool {
	return port.v != 0
}

func (port EthDev) String() string {
	if !port.Valid() {
		return "invalid"
	}
	return strconv.Itoa(port.ID())
}

// Name returns port name.
func (port EthDev) Name() string {
	var ifname [C.RTE_ETH_NAME_MAX_LEN]C.char
	res := C.rte_eth_dev_get_name_by_port(C.uint16_t(port.ID()), &ifname[0])
	if res != 0 {
		return ""
	}
	return C.GoString(&ifname[0])
}

// NumaSocket returns the NUMA socket where this EthDev is located.
func (port EthDev) NumaSocket() (socket eal.NumaSocket) {
	return eal.NumaSocketFromID(int(C.rte_eth_dev_socket_id(C.uint16_t(port.ID()))))
}

// DevInfo retrieves information about the hardware device.
func (port EthDev) DevInfo() (info DevInfo) {
	C.rte_eth_dev_info_get(C.uint16_t(port.ID()), (*C.struct_rte_eth_dev_info)(unsafe.Pointer(&info)))
	return info
}

// MacAddr retrieves MAC address of this EthDev.
func (port EthDev) MacAddr() (a EtherAddr) {
	C.rte_eth_macaddr_get(C.uint16_t(port.ID()), a.ptr())
	return a
}

// Mtu retrieves MTU of this EthDev.
func (port EthDev) Mtu() int {
	var mtu C.uint16_t
	C.rte_eth_dev_get_mtu(C.uint16_t(port.ID()), &mtu)
	return int(mtu)
}

// IsDown determines whether this EthDev is down.
func (port EthDev) IsDown() bool {
	return bool(C.EthDev_IsDown(C.uint16_t(port.ID())))
}

// Start configures and starts this EthDev.
func (port EthDev) Start(cfg Config) error {
	info := port.DevInfo()
	if info.Max_rx_queues > 0 && len(cfg.RxQueues) > int(info.Max_rx_queues) {
		return fmt.Errorf("cannot add more than %d RX queues", info.Max_rx_queues)
	}
	if info.Max_tx_queues > 0 && len(cfg.TxQueues) > int(info.Max_tx_queues) {
		return fmt.Errorf("cannot add more than %d TX queues", info.Max_tx_queues)
	}

	if cfg.Mtu > 0 {
		if res := C.rte_eth_dev_set_mtu(C.uint16_t(port.ID()), C.uint16_t(cfg.Mtu)); res != 0 {
			return fmt.Errorf("rte_eth_dev_set_mtu(%v,%d) error %d", port, cfg.Mtu, res)
		}
	}

	conf := (*C.struct_rte_eth_conf)(cfg.Conf)
	if conf == nil {
		conf = new(C.struct_rte_eth_conf)
		conf.rxmode.max_rx_pkt_len = C.uint32_t(port.Mtu())
		if info.Tx_offload_capa&C.DEV_TX_OFFLOAD_MULTI_SEGS != 0 {
			conf.txmode.offloads = C.DEV_TX_OFFLOAD_MULTI_SEGS
		}
	}

	res := C.rte_eth_dev_configure(C.uint16_t(port.ID()), C.uint16_t(len(cfg.RxQueues)),
		C.uint16_t(len(cfg.TxQueues)), conf)
	if res < 0 {
		return fmt.Errorf("rte_eth_dev_configure(%v) error %v", port, eal.Errno(-res))
	}

	for i, qcfg := range cfg.RxQueues {
		capacity := C.uint16_t(ringbuffer.AlignCapacity(qcfg.Capacity))
		res = C.rte_eth_rx_queue_setup(C.uint16_t(port.ID()), C.uint16_t(i), capacity,
			C.uint(qcfg.Socket.ID()), (*C.struct_rte_eth_rxconf)(qcfg.Conf), (*C.struct_rte_mempool)(qcfg.RxPool.Ptr()))
		if res != 0 {
			return fmt.Errorf("rte_eth_rx_queue_setup(%v,%d) error %v", port, i, eal.Errno(-res))
		}
	}

	for i, qcfg := range cfg.TxQueues {
		capacity := C.uint16_t(ringbuffer.AlignCapacity(qcfg.Capacity))
		res = C.rte_eth_tx_queue_setup(C.uint16_t(port.ID()), C.uint16_t(i), capacity,
			C.uint(qcfg.Socket.ID()), (*C.struct_rte_eth_txconf)(qcfg.Conf))
		if res != 0 {
			return fmt.Errorf("rte_eth_tx_queue_setup(%v,%d) error %d", port, i, res)
		}
	}

	if cfg.Promisc {
		C.rte_eth_promiscuous_enable(C.uint16_t(port.ID()))
	} else {
		C.rte_eth_promiscuous_disable(C.uint16_t(port.ID()))
	}

	res = C.rte_eth_dev_start(C.uint16_t(port.ID()))
	if res != 0 {
		return fmt.Errorf("rte_eth_dev_start(%v) error %d", port, res)
	}
	return nil
}

// Stop stops this EthDev.
// If mode is StopDetach, this EthDev cannot be restarted.
// Otherwise, it may be re-configured and started again.
func (port EthDev) Stop(mode StopMode) {
	C.rte_eth_dev_stop(C.uint16_t(port.ID()))
	switch mode {
	case StopDetach:
		C.rte_eth_dev_close(C.uint16_t(port.ID()))
	case StopReset:
		C.rte_eth_dev_reset(C.uint16_t(port.ID()))
	}
}

// Config contains EthDev configuration.
type Config struct {
	RxQueues []RxQueueConfig
	TxQueues []TxQueueConfig
	Mtu      int            // if non-zero, change MTU
	Promisc  bool           // promiscuous mode
	Conf     unsafe.Pointer // pointer to rte_eth_conf, nil means default
}

// AddRxQueues adds RxQueueConfig for several queues
func (cfg *Config) AddRxQueues(count int, qcfg RxQueueConfig) {
	for i := 0; i < count; i++ {
		cfg.RxQueues = append(cfg.RxQueues, qcfg)
	}
}

// AddTxQueues adds TxQueueConfig for several queues
func (cfg *Config) AddTxQueues(count int, qcfg TxQueueConfig) {
	for i := 0; i < count; i++ {
		cfg.TxQueues = append(cfg.TxQueues, qcfg)
	}
}

// RxQueueConfig contains EthDev RX queue configuration.
type RxQueueConfig struct {
	Capacity int            // ring capacity
	Socket   eal.NumaSocket // where to allocate the ring
	RxPool   *pktmbuf.Pool  // where to store packets
	Conf     unsafe.Pointer // pointer to rte_eth_rxconf
}

// TxQueueConfig contains EthDev TX queue configuration.
type TxQueueConfig struct {
	Capacity int            // ring capacity
	Socket   eal.NumaSocket // where to allocate the ring
	Conf     unsafe.Pointer // pointer to rte_eth_txconf
}

// StopMode selects the behavior of stopping an EthDev.
type StopMode int

const (
	// StopDetach detaches the device. It cannot be restarted.
	StopDetach StopMode = iota

	// StopReset resets the device. It can be restarted.
	StopReset
)
