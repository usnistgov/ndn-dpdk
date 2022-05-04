package ethdev

/*
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

// RxQueue represents an RX queue.
type RxQueue struct {
	Port  uint16
	Queue uint16
}

// Info returns information about the RX queue.
func (q RxQueue) Info() (info RxqInfo) {
	C.rte_eth_rx_queue_info_get(C.uint16_t(q.Port), C.uint16_t(q.Queue), (*C.struct_rte_eth_rxq_info)(unsafe.Pointer(&info.RxqInfoC)))
	info.q = q
	return
}

// RxBurst receives a burst of input packets.
// Returns the number of packets received and written into vec.
func (q RxQueue) RxBurst(vec pktmbuf.Vector) int {
	if len(vec) == 0 {
		return 0
	}
	res := C.rte_eth_rx_burst(C.uint16_t(q.Port), C.uint16_t(q.Queue),
		cptr.FirstPtr[*C.struct_rte_mbuf](vec), C.uint16_t(len(vec)))
	return int(res)
}

// RxqInfo provides contextual information of an RX queue.
type RxqInfo struct {
	RxqInfoC
	q RxQueue
}

// BurstMode retrieves queue burst mode.
func (info RxqInfo) BurstMode() BurstModeInfo {
	var bm C.struct_rte_eth_burst_mode
	res := C.rte_eth_rx_burst_mode_get(C.uint16_t(info.q.Port), C.uint16_t(info.q.Queue), &bm)
	return burstModeInfoFromC(res, bm)
}

// MarshalJSON implements json.Marshaler interface.
func (info RxqInfo) MarshalJSON() ([]byte, error) {
	return infoJSON(info, info.RxqInfoC)
}

// TxQueue represents a TX queue.
type TxQueue struct {
	Port  uint16
	Queue uint16
}

// Info returns information about the TX queue.
func (q TxQueue) Info() (info TxqInfo) {
	C.rte_eth_tx_queue_info_get(C.uint16_t(q.Port), C.uint16_t(q.Queue), (*C.struct_rte_eth_txq_info)(unsafe.Pointer(&info.TxqInfoC)))
	info.q = q
	return
}

// TxBurst transmits a burst of output packets.
// Returns the number of packets enqueued.
func (q TxQueue) TxBurst(vec pktmbuf.Vector) int {
	return int(C.rte_eth_tx_burst(C.uint16_t(q.Port), C.uint16_t(q.Queue),
		cptr.FirstPtr[*C.struct_rte_mbuf](vec), C.uint16_t(len(vec))))
}

// TxqInfo provides contextual information of a TX queue.
type TxqInfo struct {
	TxqInfoC
	q TxQueue
}

// BurstMode retrieves queue burst mode.
func (info TxqInfo) BurstMode() BurstModeInfo {
	var bm C.struct_rte_eth_burst_mode
	res := C.rte_eth_tx_burst_mode_get(C.uint16_t(info.q.Port), C.uint16_t(info.q.Queue), &bm)
	return burstModeInfoFromC(res, bm)
}

// MarshalJSON implements json.Marshaler interface.
func (info TxqInfo) MarshalJSON() ([]byte, error) {
	return infoJSON(info, info.TxqInfoC)
}

// BurstModeInfo describes queue burst mode.
type BurstModeInfo struct {
	Flags uint64 `json:"flags"`
	Info  string `json:"info"`
}

func burstModeInfoFromC(res C.int, bm C.struct_rte_eth_burst_mode) (info BurstModeInfo) {
	if res == 0 {
		info.Flags = uint64(bm.flags)
		info.Info = C.GoString(&bm.info[0])
	} else {
		info.Info = fmt.Sprint("ERROR: ", eal.MakeErrno(res))
	}
	return
}
