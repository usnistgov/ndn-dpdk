//go:build ignore

package ethdev

/*
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"

// DescLim contains information about hardware descriptor ring limitations.
type DescLim C.struct_rte_eth_desc_lim

// StatsBasic contains basic statistics for an Ethernet port.
type StatsBasic C.struct_rte_eth_stats

type DevInfoC C.struct_rte_eth_dev_info
type PortConfC C.struct_rte_eth_dev_portconf
type RxConfC C.struct_rte_eth_rxconf
type RxqInfoC C.struct_rte_eth_rxq_info
type RxsegCapaC C.struct_rte_eth_rxseg_capa
type SwitchInfoC C.struct_rte_eth_switch_info
type ThreshC C.struct_rte_eth_thresh
type TxConfC C.struct_rte_eth_txconf
type TxqInfoC C.struct_rte_eth_txq_info
