package dpdk

/*
#include "ethdev.h"
#include <rte_eth_ring.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// EthDev represents an Ethernet adapter.
type EthDev struct {
	v int // ethdev ID + 1
}

// EthDevFromID converts port ID to EthDev.
func EthDevFromID(id int) (port EthDev) {
	if id < 0 || id >= C.RTE_MAX_ETHPORTS {
		return port
	}
	port.v = id + 1
	return port
}

// ListEthDevs returns a list of Ethernet adapters.
func ListEthDevs() (list []EthDev) {
	for p := C.rte_eth_find_next(0); p < C.RTE_MAX_ETHPORTS; p = C.rte_eth_find_next(p + 1) {
		list = append(list, EthDevFromID(int(p)))
	}
	return list
}

// FindEthDev locates an EthDev by name.
func FindEthDev(name string) EthDev {
	for p := C.rte_eth_find_next(0); p < C.RTE_MAX_ETHPORTS; p = C.rte_eth_find_next(p + 1) {
		port := EthDevFromID(int(p))
		if port.GetName() == name {
			return port
		}
	}
	return EthDev{}
}

// NewEthDevFromRings creates an EthDev using net/ring driver.
func NewEthDevFromRings(name string, rxRings []Ring, txRings []Ring, socket NumaSocket) (dev EthDev, e error) {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	res := C.rte_eth_from_rings(nameC,
		(**C.struct_rte_ring)(unsafe.Pointer(&rxRings[0])), C.unsigned(len(rxRings)),
		(**C.struct_rte_ring)(unsafe.Pointer(&txRings[0])), C.unsigned(len(txRings)),
		C.unsigned(socket.ID()))
	if res < 0 {
		return EthDev{}, GetErrno()
	}
	return EthDevFromID(int(res)), nil
}

// ID returns EthDev ID.
func (port EthDev) ID() int {
	return port.v - 1
}

// IsValid returns true if this is a valid Ethernet port.
func (port EthDev) IsValid() bool {
	return port.v != 0
}

// GetName returns port name.
func (port EthDev) GetName() string {
	var ifname [C.RTE_ETH_NAME_MAX_LEN]C.char
	res := C.rte_eth_dev_get_name_by_port(C.uint16_t(port.ID()), &ifname[0])
	if res != 0 {
		return ""
	}
	return C.GoString(&ifname[0])
}

// GetNumaSocket returns the NUMA socket where this EthDev is located.
func (port EthDev) GetNumaSocket() (socket NumaSocket) {
	return NumaSocketFromID(int(C.rte_eth_dev_socket_id(C.uint16_t(port.ID()))))
}

// GetDevInfo retrieves information about the hardware device.
func (port EthDev) GetDevInfo() (info EthDevInfo) {
	C.rte_eth_dev_info_get(C.uint16_t(port.ID()), (*C.struct_rte_eth_dev_info)(unsafe.Pointer(&info)))
	return info
}

// Configure configures this EthDev.
func (port EthDev) Configure(cfg EthDevConfig) (rxQueues []EthRxQueue, txQueues []EthTxQueue, e error) {
	portId := C.uint16_t(port.ID())
	if cfg.Mtu > 0 {
		if res := C.rte_eth_dev_set_mtu(C.uint16_t(port.ID()), C.uint16_t(cfg.Mtu)); res != 0 {
			return nil, nil, fmt.Errorf("rte_eth_dev_set_mtu(%d,%d) error code %d", port, cfg.Mtu, res)
		}
	}

	conf := (*C.struct_rte_eth_conf)(cfg.Conf)
	if conf == nil {
		conf = new(C.struct_rte_eth_conf)
		conf.rxmode.max_rx_pkt_len = C.uint32_t(port.GetMtu())
		if info := port.GetDevInfo(); info.Tx_offload_capa&C.DEV_TX_OFFLOAD_MULTI_SEGS != 0 {
			conf.txmode.offloads = C.DEV_TX_OFFLOAD_MULTI_SEGS
		}
	}

	res := C.rte_eth_dev_configure(portId, C.uint16_t(len(cfg.RxQueues)),
		C.uint16_t(len(cfg.TxQueues)), conf)
	if res < 0 {
		return nil, nil, fmt.Errorf("rte_eth_dev_configure(%d) error code %d", port, res)
	}

	rxQueues = make([]EthRxQueue, len(cfg.RxQueues))
	for i, qcfg := range cfg.RxQueues {
		res = C.rte_eth_rx_queue_setup(portId, C.uint16_t(i), C.uint16_t(qcfg.Capacity),
			C.uint(qcfg.Socket.ID()), (*C.struct_rte_eth_rxconf)(qcfg.Conf), qcfg.Mp.c)
		if res != 0 {
			return nil, nil, fmt.Errorf("rte_eth_rx_queue_setup(%d,%d) error %d", port, i, res)
		}
		rxQueues[i].port = portId
		rxQueues[i].queue = C.uint16_t(i)
	}

	txQueues = make([]EthTxQueue, len(cfg.TxQueues))
	for i, qcfg := range cfg.TxQueues {
		res = C.rte_eth_tx_queue_setup(portId, C.uint16_t(i), C.uint16_t(qcfg.Capacity),
			C.uint(qcfg.Socket.ID()), (*C.struct_rte_eth_txconf)(qcfg.Conf))
		if res != 0 {
			return nil, nil, fmt.Errorf("rte_eth_tx_queue_setup(%d,%d) error %d", port, i, res)
		}
		txQueues[i].port = portId
		txQueues[i].queue = C.uint16_t(i)
	}

	return rxQueues, txQueues, nil
}

// GetMacAddress retrieves MAC address of this EthDev.
func (port EthDev) GetMacAddr() (a EtherAddr) {
	C.rte_eth_macaddr_get(C.uint16_t(port.ID()), a.getPtr())
	return a
}

// GetMtu retrieves MTU of this EthDev.
func (port EthDev) GetMtu() int {
	var mtu C.uint16_t
	C.rte_eth_dev_get_mtu(C.uint16_t(port.ID()), &mtu)
	return int(mtu)
}

// SetMtu updates MTU of this EthDev.
func (port EthDev) SetMtu(mtu int) error {
	res := C.rte_eth_dev_set_mtu(C.uint16_t(port.ID()), C.uint16_t(mtu))
	if res != 0 {
		return Errno(-res)
	}
	return nil
}

// IsPromiscuous determins whether this EthDev is operating in promiscuous mode.
func (port EthDev) IsPromiscuous() (bool, error) {
	res := C.rte_eth_promiscuous_get(C.uint16_t(port.ID()))
	switch res {
	case 1:
		return true, nil
	case 0:
		return false, nil
	default:
		return false, fmt.Errorf("rte_eth_promiscuous_get(%d) error", port)
	}
}

// SetPromiscuous updates promiscuous mode setting.
func (port EthDev) SetPromiscuous(enable bool) {
	if enable {
		C.rte_eth_promiscuous_enable(C.uint16_t(port.ID()))
	} else {
		C.rte_eth_promiscuous_disable(C.uint16_t(port.ID()))
	}
}

// IsDown determines whether this EthDev is down.
func (port EthDev) IsDown() bool {
	return bool(C.EthDev_IsDown(C.uint16_t(port.ID())))
}

// Start starts this EthDev.
// The EthDev should be configured.
func (port EthDev) Start() error {
	res := C.rte_eth_dev_start(C.uint16_t(port.ID()))
	if res != 0 {
		return fmt.Errorf("rte_eth_dev_start(%d) error %d", port, res)
	}
	return nil
}

// Stop stops this EthDev.
func (port EthDev) Stop() {
	C.rte_eth_dev_stop(C.uint16_t(port.ID()))
}

// Close closes this EthDev.
// It cannot be restarted.
func (port EthDev) Close() error {
	C.rte_eth_dev_close(C.uint16_t(port.ID()))
	return nil
}

// Reset re-initializes this EthDev.
// It can be configured and started again.
func (port EthDev) Reset() error {
	res := C.rte_eth_dev_reset(C.uint16_t(port.ID()))
	if res != 0 {
		return Errno(-res)
	}
	return nil
}

// GetStats retrieves hardware statistics.
func (port EthDev) GetStats() (es EthStats) {
	C.rte_eth_stats_get(C.uint16_t(port.ID()), (*C.struct_rte_eth_stats)(unsafe.Pointer(&es)))
	return es
}

// ResetStats clears hardware statistics.
func (port EthDev) ResetStats() {
	C.rte_eth_stats_reset(C.uint16_t(port.ID()))
}

// EthDevConfig contains EthDev configuration.
type EthDevConfig struct {
	RxQueues []EthRxQueueConfig
	TxQueues []EthTxQueueConfig
	Mtu      int            // if non-zero, change MTU
	Conf     unsafe.Pointer // pointer to rte_eth_conf, nil means default
}

// EthRxQueueConfig contains EthDev RX queue configuration.
type EthRxQueueConfig struct {
	Capacity int
	Socket   NumaSocket     // where to allocate the ring
	Mp       PktmbufPool    // where to store packets
	Conf     unsafe.Pointer // pointer to rte_eth_rxconf
}

// EthRxQueueConfig contains EthDev TX queue configuration.
type EthTxQueueConfig struct {
	Capacity int
	Socket   NumaSocket     // where to allocate the ring
	Conf     unsafe.Pointer // pointer to rte_eth_txconf
}

// EthRxQueue represents an RX queue.
type EthRxQueue struct {
	port  C.uint16_t
	queue C.uint16_t
}

// RxBurst receives a burst of input packets.
// Returns the number of packets received and written into pkts.
func (q EthRxQueue) RxBurst(pkts []Packet) int {
	res := C.rte_eth_rx_burst(q.port, q.queue, (**C.struct_rte_mbuf)(unsafe.Pointer(&pkts[0])),
		C.uint16_t(len(pkts)))
	return int(res)
}

// EthRxQueue represents an TX queue.
type EthTxQueue struct {
	port  C.uint16_t
	queue C.uint16_t
}

// TxBurst transmits a burst of output packets.
// Returns the number of packets enqueued.
func (q EthTxQueue) TxBurst(pkts []Packet) int {
	if len(pkts) == 0 {
		return 0
	}
	res := C.rte_eth_tx_burst(q.port, q.queue, (**C.struct_rte_mbuf)(unsafe.Pointer(&pkts[0])),
		C.uint16_t(len(pkts)))
	return int(res)
}

func (es EthStats) String() string {
	return fmt.Sprintf("RX %d pkts, %d bytes, %d missed, %d errors, %d nombuf; TX %d pkts, %d bytes, %d errors",
		es.Ipackets, es.Ibytes, es.Imissed, es.Ierrors, es.Rx_nombuf, es.Opackets, es.Obytes, es.Oerrors)
}
