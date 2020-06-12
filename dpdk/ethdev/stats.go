package ethdev

/*
#include "ethdev.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// GetStats retrieves hardware statistics.
func (port EthDev) GetStats() (es Stats) {
	C.rte_eth_stats_get(C.uint16_t(port.ID()), (*C.struct_rte_eth_stats)(unsafe.Pointer(&es)))
	return es
}

// ResetStats clears hardware statistics.
func (port EthDev) ResetStats() {
	C.rte_eth_stats_reset(C.uint16_t(port.ID()))
}

func (es Stats) String() string {
	return fmt.Sprintf("RX %d pkts, %d bytes, %d missed, %d errors, %d nombuf; TX %d pkts, %d bytes, %d errors",
		es.Ipackets, es.Ibytes, es.Imissed, es.Ierrors, es.Rx_nombuf, es.Opackets, es.Obytes, es.Oerrors)
}
