package ethdev

import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

// Config contains EthDev configuration.
type Config struct {
	RxQueues []RxQueueConfig
	TxQueues []TxQueueConfig
	MTU      int  // if non-zero, change MTU
	Promisc  bool // promiscuous mode
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
