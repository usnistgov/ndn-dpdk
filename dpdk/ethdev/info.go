package ethdev

/*
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"
import (
	"fmt"

	"github.com/pkg/math"
)

const (
	txOffloadMultiSegs = C.DEV_TX_OFFLOAD_MULTI_SEGS
	txOffloadChecksum  = C.DEV_TX_OFFLOAD_IPV4_CKSUM | C.DEV_TX_OFFLOAD_UDP_CKSUM
)

// HasTxMultiSegOffload determines whether device can transmit multi-segment packets.
func (info DevInfo) HasTxMultiSegOffload() bool {
	return (info.Tx_offload_capa & txOffloadMultiSegs) == txOffloadMultiSegs
}

// HasTxChecksumOffload determines whether device can compute IPv4 and UDP checksum offload upon transmission.
func (info DevInfo) HasTxChecksumOffload() bool {
	return (info.Tx_offload_capa & txOffloadChecksum) == txOffloadChecksum
}

// adjustQueueCapacity adjust RX/TX queue capacity to satisfy driver requirements.
func (lim DescLim) adjustQueueCapacity(capacity int) int {
	capacity -= capacity % int(lim.Align)
	return math.MinInt(math.MaxInt(int(lim.Min), capacity), int(lim.Max))
}

func (stats Stats) String() string {
	return fmt.Sprintf("RX %d pkts, %d bytes, %d missed, %d errors, %d nombuf; TX %d pkts, %d bytes, %d errors",
		stats.Ipackets, stats.Ibytes, stats.Imissed, stats.Ierrors, stats.Rx_nombuf, stats.Opackets, stats.Obytes, stats.Oerrors)
}
