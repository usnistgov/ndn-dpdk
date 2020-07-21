// +build ignore

package ethdev

/*
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"

// Contextual information of an Ethernet port.
type DevInfo C.struct_rte_eth_dev_info

// Statistics for an Ethernet port.
type Stats C.struct_rte_eth_stats
