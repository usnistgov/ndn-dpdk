//go:build ignore

package ethdev

/*
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"

type DevInfoC C.struct_rte_eth_dev_info

// DescLim contains information about hardware descriptor ring limitations.
type DescLim C.struct_rte_eth_desc_lim

type ThreshC C.struct_rte_eth_thresh

type RxConfC C.struct_rte_eth_rxconf

type RxqInfoC C.struct_rte_eth_rxq_info

type TxConfC C.struct_rte_eth_txconf

type TxqInfoC C.struct_rte_eth_txq_info

// Stats contains statistics for an Ethernet port.
type Stats C.struct_rte_eth_stats
