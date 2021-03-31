package ethdev

/*
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/pkg/math"
)

const (
	txOffloadMultiSegs = C.DEV_TX_OFFLOAD_MULTI_SEGS
	txOffloadChecksum  = C.DEV_TX_OFFLOAD_IPV4_CKSUM | C.DEV_TX_OFFLOAD_UDP_CKSUM
)

// DriverName returns DPDK net driver name.
func (info DevInfo) DriverName() string {
	return C.GoString((*C.char)(unsafe.Pointer(info.Driver_name)))
}

// IsVDev determines whether the driver is a virtual device.
func (info DevInfo) IsVDev() bool {
	switch info.DriverName() {
	case "net_af_packet", "net_af_xdp", "net_memif", "net_ring":
		return true
	}
	return false
}

// HasTxMultiSegOffload determines whether device can transmit multi-segment packets.
func (info DevInfo) HasTxMultiSegOffload() bool {
	if (info.Tx_offload_capa & txOffloadMultiSegs) == txOffloadMultiSegs {
		return true
	}

	switch info.DriverName() { // some drivers support multi-segment TX but do not advertise it
	case "net_memif", "net_ring":
		return true
	}
	return false
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
