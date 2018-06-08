package dpdk

/*
#include "ethdev.h"
#include <rte_eth_ring.h>
*/
import "C"
import (
	"fmt"
	"net"
	"unsafe"
)

type EthDev uint16

const ETHDEV_INVALID = EthDev(0xFFFF)

func ListEthDevs() []EthDev {
	var l []EthDev
	for p := C.rte_eth_find_next(0); p < C.RTE_MAX_ETHPORTS; p = C.rte_eth_find_next(p + 1) {
		l = append(l, EthDev(p))
	}
	return l
}

func FindEthDev(name string) EthDev {
	for p := C.rte_eth_find_next(0); p < C.RTE_MAX_ETHPORTS; p = C.rte_eth_find_next(p + 1) {
		port := EthDev(p)
		if port.GetName() == name {
			return port
		}
	}
	return ETHDEV_INVALID
}

func NewEthDevFromRings(name string, rxRings []Ring, txRings []Ring, socket NumaSocket) (dev EthDev, e error) {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	res := C.rte_eth_from_rings(nameC,
		(**C.struct_rte_ring)(unsafe.Pointer(&rxRings[0])), C.unsigned(len(rxRings)),
		(**C.struct_rte_ring)(unsafe.Pointer(&txRings[0])), C.unsigned(len(txRings)),
		C.unsigned(socket))
	if res < 0 {
		return ETHDEV_INVALID, GetErrno()
	}
	return EthDev(res), nil
}

func (port EthDev) IsValid() bool {
	return port != ETHDEV_INVALID
}

func (port EthDev) GetName() string {
	var ifname [C.RTE_ETH_NAME_MAX_LEN]C.char
	res := C.rte_eth_dev_get_name_by_port(C.uint16_t(port), &ifname[0])
	if res != 0 {
		return ""
	}
	return C.GoString(&ifname[0])
}

func (port EthDev) GetNumaSocket() NumaSocket {
	return NumaSocket(C.rte_eth_dev_socket_id(C.uint16_t(port)))
}

type EthDevConfig struct {
	RxQueues []EthRxQueueConfig
	TxQueues []EthTxQueueConfig
	Conf     unsafe.Pointer // pointer to rte_eth_conf, nil means default
}

func (cfg *EthDevConfig) AddRxQueue(qcfg EthRxQueueConfig) {
	cfg.RxQueues = append(cfg.RxQueues, qcfg)
}

func (cfg *EthDevConfig) AddTxQueue(qcfg EthTxQueueConfig) {
	cfg.TxQueues = append(cfg.TxQueues, qcfg)
}

type EthRxQueueConfig struct {
	Capacity int
	Socket   NumaSocket     // where to allocate the ring
	Mp       PktmbufPool    // where to store packets
	Conf     unsafe.Pointer // pointer to rte_eth_rxconf
}

type EthTxQueueConfig struct {
	Capacity int
	Socket   NumaSocket     // where to allocate the ring
	Conf     unsafe.Pointer // pointer to rte_eth_txconf
}

func (port EthDev) Configure(cfg EthDevConfig) (rxQueues []EthRxQueue, txQueues []EthTxQueue, e error) {
	portId := C.uint16_t(port)
	var defaultConf C.struct_rte_eth_conf
	defaultConf.rxmode.max_rx_pkt_len = C.ETHER_MAX_LEN
	conf := (*C.struct_rte_eth_conf)(cfg.Conf)
	if conf == nil {
		conf = &defaultConf
	}

	res := C.rte_eth_dev_configure(portId, C.uint16_t(len(cfg.RxQueues)),
		C.uint16_t(len(cfg.TxQueues)), conf)
	if res < 0 {
		return nil, nil, fmt.Errorf("rte_eth_dev_configure(%d) error code %d", port, res)
	}

	rxQueues = make([]EthRxQueue, len(cfg.RxQueues))
	for i, qcfg := range cfg.RxQueues {
		res = C.rte_eth_rx_queue_setup(portId, C.uint16_t(i), C.uint16_t(qcfg.Capacity),
			C.uint(qcfg.Socket), (*C.struct_rte_eth_rxconf)(qcfg.Conf), qcfg.Mp.c)
		if res != 0 {
			return nil, nil, fmt.Errorf("rte_eth_rx_queue_setup(%d,%d) error %d", port, i, res)
		}
		rxQueues[i].port = portId
		rxQueues[i].queue = C.uint16_t(i)
	}

	txQueues = make([]EthTxQueue, len(cfg.TxQueues))
	for i, qcfg := range cfg.TxQueues {
		res = C.rte_eth_tx_queue_setup(portId, C.uint16_t(i), C.uint16_t(qcfg.Capacity),
			C.uint(qcfg.Socket), (*C.struct_rte_eth_txconf)(qcfg.Conf))
		if res != 0 {
			return nil, nil, fmt.Errorf("rte_eth_tx_queue_setup(%d,%d) error %d", port, i, res)
		}
		txQueues[i].port = portId
		txQueues[i].queue = C.uint16_t(i)
	}

	return rxQueues, txQueues, nil
}

func (port EthDev) GetMacAddr() net.HardwareAddr {
	var macAddr C.struct_ether_addr
	C.rte_eth_macaddr_get(C.uint16_t(port), &macAddr)
	return net.HardwareAddr(C.GoBytes(unsafe.Pointer(&macAddr.addr_bytes[0]), C.ETHER_ADDR_LEN))
}

func (port EthDev) GetMtu() int {
	var mtu C.uint16_t
	C.rte_eth_dev_get_mtu(C.uint16_t(port), &mtu)
	return int(mtu)
}

func (port EthDev) SetMtu(mtu int) error {
	res := C.rte_eth_dev_set_mtu(C.uint16_t(port), C.uint16_t(mtu))
	if res != 0 {
		return Errno(-res)
	}
	return nil
}

func (port EthDev) IsPromiscuous() (bool, error) {
	res := C.rte_eth_promiscuous_get(C.uint16_t(port))
	switch res {
	case 1:
		return true, nil
	case 0:
		return false, nil
	default:
		return false, fmt.Errorf("rte_eth_promiscuous_get(%d) error", port)
	}
}

func (port EthDev) SetPromiscuous(enable bool) {
	if enable {
		C.rte_eth_promiscuous_enable(C.uint16_t(port))
	} else {
		C.rte_eth_promiscuous_disable(C.uint16_t(port))
	}
}

func (port EthDev) IsDown() bool {
	return bool(C.EthDev_IsDown(C.uint16_t(port)))
}

func (port EthDev) Start() error {
	res := C.rte_eth_dev_start(C.uint16_t(port))
	if res != 0 {
		return fmt.Errorf("rte_eth_dev_start(%d) error %d", port, res)
	}
	return nil
}

func (port EthDev) Stop() {
	C.rte_eth_dev_stop(C.uint16_t(port))
}

func (port EthDev) Close(detach bool) error {
	C.rte_eth_dev_close(C.uint16_t(port))
	if detach {
		var devname [C.RTE_DEV_NAME_MAX_LEN]C.char
		res := C.rte_eth_dev_detach(C.uint16_t(port), &devname[0])
		if res != 0 {
			return fmt.Errorf("rte_eth_dev_detach(%d) error %d", port, res)
		}
	}
	return nil
}

type EthRxQueue struct {
	port  C.uint16_t
	queue C.uint16_t
}

func (q EthRxQueue) GetPort() uint16 {
	return uint16(q.port)
}

func (q EthRxQueue) GetQueue() uint16 {
	return uint16(q.queue)
}

// Retrieve a burst of input packets.
// Return the number of packets received and written into pkts.
func (q EthRxQueue) RxBurst(pkts []Packet) int {
	res := C.rte_eth_rx_burst(q.port, q.queue, (**C.struct_rte_mbuf)(unsafe.Pointer(&pkts[0])),
		C.uint16_t(len(pkts)))
	return int(res)
}

type EthTxQueue struct {
	port  C.uint16_t
	queue C.uint16_t
}

func (q EthTxQueue) GetPort() uint16 {
	return uint16(q.port)
}

func (q EthTxQueue) GetQueue() uint16 {
	return uint16(q.queue)
}

// Send a burst of output packets.
// Return the number of packets enqueued.
func (q EthTxQueue) TxBurst(pkts []Packet) int {
	if len(pkts) == 0 {
		return 0
	}
	res := C.rte_eth_tx_burst(q.port, q.queue, (**C.struct_rte_mbuf)(unsafe.Pointer(&pkts[0])),
		C.uint16_t(len(pkts)))
	return int(res)
}

func (port EthDev) GetStats() (es EthStats) {
	C.rte_eth_stats_get(C.uint16_t(port), (*C.struct_rte_eth_stats)(unsafe.Pointer(&es)))
	return es
}

func (port EthDev) ResetStats() {
	C.rte_eth_stats_reset(C.uint16_t(port))
}
