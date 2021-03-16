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
	"sync"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go.uber.org/zap"
)

var randomizedMacAddrs sync.Map

// EthDev represents an Ethernet adapter.
type EthDev struct {
	v int // ethdev ID + 1
}

// ID returns EthDev ID.
func (port EthDev) ID() int {
	return port.v - 1
}

func (port EthDev) cID() C.uint16_t {
	return C.uint16_t(port.ID())
}

// Valid returns true if this is a valid Ethernet port.
func (port EthDev) Valid() bool {
	return port.v != 0
}

// ZapField returns a zap.Field for logging.
func (port EthDev) ZapField(key string) zap.Field {
	if !port.Valid() {
		return zap.String(key, "invalid")
	}
	return zap.Int(key, port.ID())
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
	res := C.rte_eth_dev_get_name_by_port(port.cID(), &ifname[0])
	if res != 0 {
		return ""
	}
	return C.GoString(&ifname[0])
}

// NumaSocket returns the NUMA socket where this EthDev is located.
func (port EthDev) NumaSocket() (socket eal.NumaSocket) {
	return eal.NumaSocketFromID(int(C.rte_eth_dev_socket_id(port.cID())))
}

// DevInfo retrieves information about the hardware device.
func (port EthDev) DevInfo() (info DevInfo) {
	C.rte_eth_dev_info_get(port.cID(), (*C.struct_rte_eth_dev_info)(unsafe.Pointer(&info)))
	return info
}

// MacAddr retrieves MAC address of this EthDev.
// If the underlying EthDev returns an invalid MAC address, a random MAC address is returned instead.
func (port EthDev) MacAddr() (a net.HardwareAddr) {
	var c C.struct_rte_ether_addr
	C.rte_eth_macaddr_get(port.cID(), &c)
	a = net.HardwareAddr(C.GoBytes(unsafe.Pointer(&c.addr_bytes[0]), C.RTE_ETHER_ADDR_LEN))
	if !macaddr.IsUnicast(a) {
		randomized, _ := randomizedMacAddrs.LoadOrStore(port.ID(), macaddr.MakeRandom(false))
		a = randomized.(net.HardwareAddr)
	}
	return a
}

// MTU retrieves MTU of this EthDev.
func (port EthDev) MTU() int {
	var mtu C.uint16_t
	C.rte_eth_dev_get_mtu(port.cID(), &mtu)
	return int(mtu)
}

// IsDown determines whether this EthDev is down.
func (port EthDev) IsDown() bool {
	return bool(C.EthDev_IsDown(port.cID()))
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

	if cfg.MTU > 0 {
		if res := C.rte_eth_dev_set_mtu(port.cID(), C.uint16_t(cfg.MTU)); res != 0 {
			return fmt.Errorf("rte_eth_dev_set_mtu(%v,%d) error %w", port, cfg.MTU, eal.Errno(-res))
		}
	}

	conf := (*C.struct_rte_eth_conf)(cfg.Conf)
	if conf == nil {
		conf = &C.struct_rte_eth_conf{}
		conf.rxmode.max_rx_pkt_len = C.uint32_t(port.MTU())
		conf.txmode.offloads = C.uint64_t(info.Tx_offload_capa & (txOffloadMultiSegs | txOffloadChecksum))
	}

	res := C.rte_eth_dev_configure(port.cID(), C.uint16_t(len(cfg.RxQueues)), C.uint16_t(len(cfg.TxQueues)), conf)
	if res < 0 {
		return fmt.Errorf("rte_eth_dev_configure(%v) error %w", port, eal.Errno(-res))
	}

	for i, qcfg := range cfg.RxQueues {
		capacity := info.Rx_desc_lim.adjustQueueCapacity(qcfg.Capacity)
		res = C.rte_eth_rx_queue_setup(port.cID(), C.uint16_t(i), C.uint16_t(capacity),
			C.uint(qcfg.Socket.ID()), (*C.struct_rte_eth_rxconf)(qcfg.Conf), (*C.struct_rte_mempool)(qcfg.RxPool.Ptr()))
		if res != 0 {
			return fmt.Errorf("rte_eth_rx_queue_setup(%v,%d) error %w", port, i, eal.Errno(-res))
		}
	}

	for i, qcfg := range cfg.TxQueues {
		capacity := info.Tx_desc_lim.adjustQueueCapacity(qcfg.Capacity)
		res = C.rte_eth_tx_queue_setup(port.cID(), C.uint16_t(i), C.uint16_t(capacity),
			C.uint(qcfg.Socket.ID()), (*C.struct_rte_eth_txconf)(qcfg.Conf))
		if res != 0 {
			return fmt.Errorf("rte_eth_tx_queue_setup(%v,%d) error %w", port, i, eal.Errno(-res))
		}
	}

	if cfg.Promisc {
		C.rte_eth_promiscuous_enable(port.cID())
	} else {
		C.rte_eth_promiscuous_disable(port.cID())
	}

	res = C.rte_eth_dev_start(port.cID())
	if res != 0 {
		return fmt.Errorf("rte_eth_dev_start(%v) error %w", port, eal.Errno(-res))
	}
	return nil
}

// Stop stops this EthDev.
// If mode is StopDetach, this EthDev cannot be restarted.
// Otherwise, it may be re-configured and started again.
func (port EthDev) Stop(mode StopMode) error {
	res := C.rte_eth_dev_stop(port.cID())
	if res != 0 {
		return eal.Errno(-res)
	}

	switch mode {
	case StopDetach:
		res = C.rte_eth_dev_close(port.cID())
	case StopReset:
		res = C.rte_eth_dev_reset(port.cID())
	}
	if res != 0 {
		return eal.Errno(-res)
	}
	return nil
}

// Stats retrieves hardware statistics.
func (port EthDev) Stats() (stats Stats) {
	C.rte_eth_stats_get(port.cID(), (*C.struct_rte_eth_stats)(unsafe.Pointer(&stats)))
	return
}

// ResetStats clears hardware statistics.
func (port EthDev) ResetStats() {
	C.rte_eth_stats_reset(port.cID())
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
