// +build ignore

package ethdev

/*
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"

// DevInfo provides contextual information of an Ethernet port.
type DevInfo C.struct_rte_eth_dev_info

// DescLim contains information about hardware descriptor ring limitations.
type DescLim C.struct_rte_eth_desc_lim

// Stats contains statistics for an Ethernet port.
type Stats C.struct_rte_eth_stats
