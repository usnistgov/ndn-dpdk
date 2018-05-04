// +build ignore

package dpdk

/*
#include "ethdev.h"
*/
import "C"

import (
	"fmt"
)

// Statistics for an Ethernet port.
type EthStats C.struct_rte_eth_stats

func (es EthStats) String() string {
	return fmt.Sprintf("RX %d pkts, %d bytes, %d missed, %d errors, %d nombuf; TX %d pkts, %d bytes, %d errors",
		es.Ipackets, es.Ibytes, es.Imissed, es.Ierrors, es.Rx_nombuf, es.Opackets, es.Obytes, es.Oerrors)
}
