package dpdk

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk

#include <rte_config.h>
#include <rte_ethdev.h>
#include <rte_ether.h>
*/
import "C"
import (
	"fmt"
	"net"
	"unsafe"
)

type EthDev uint16

func ListEthDevs() []EthDev {
	var l []EthDev
	for p := C.rte_eth_find_next(0); p < C.RTE_MAX_ETHPORTS; p = C.rte_eth_find_next(p + 1) {
		l = append(l, EthDev(p))
	}
	return l
}

func CountEthDevs() uint {
	return uint(C.rte_eth_dev_count())
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

type EthRxQueueConfig struct {
	Capacity uint
	Socket   NumaSocket     // where to allocate the ring
	Mp       PktmbufPool    // where to store packets
	Conf     unsafe.Pointer // pointer to rte_eth_rxconf
}

type EthTxQueueConfig struct {
	Capacity uint
	Socket   NumaSocket     // where to allocate the ring
	Conf     unsafe.Pointer // pointer to rte_eth_txconf
}

func (port EthDev) Configure(cfg *EthDevConfig) ([]EthRxQueue, []EthTxQueue, error) {
	portId := C.uint16_t(port)
	var emptyEthConf C.struct_rte_eth_conf
	conf := (*C.struct_rte_eth_conf)(cfg.Conf)
	if conf == nil {
		conf = &emptyEthConf
	}

	res := C.rte_eth_dev_configure(portId, C.uint16_t(len(cfg.RxQueues)),
		C.uint16_t(len(cfg.TxQueues)), conf)
	if res < 0 {
		return nil, nil, fmt.Errorf("rte_eth_dev_configure(%d) error code %d", port, res)
	}

	rxQueues := make([]EthRxQueue, len(cfg.RxQueues))
	for i, qcfg := range cfg.RxQueues {
		res = C.rte_eth_rx_queue_setup(portId, C.uint16_t(i), C.uint16_t(qcfg.Capacity),
			C.uint(qcfg.Socket), (*C.struct_rte_eth_rxconf)(qcfg.Conf), qcfg.Mp.ptr)
		if res != 0 {
			return nil, nil, fmt.Errorf("rte_eth_rx_queue_setup(%d,%d) error %d", port, i, res)
		}
		rxQueues[i].port = portId
		rxQueues[i].queue = C.uint16_t(i)
	}

	txQueues := make([]EthTxQueue, len(cfg.RxQueues))
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

func (port EthDev) GetMtu() uint {
	var mtu C.uint16_t
	C.rte_eth_dev_get_mtu(C.uint16_t(port), &mtu)
	return uint(mtu)
}

func (port EthDev) SetMtu(mtu uint) error {
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

type EthRxQueue struct {
	port  C.uint16_t
	queue C.uint16_t
}

type EthTxQueue struct {
	port  C.uint16_t
	queue C.uint16_t
}
