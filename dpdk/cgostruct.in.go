// +build ignore

package dpdk

/*
#include "ethdev.h"
#include <rte_pci.h>
*/
import "C"

// Contextual information of an Ethernet port.
type EthDevInfo C.struct_rte_eth_dev_info

// Statistics for an Ethernet port.
type EthStats C.struct_rte_eth_stats

// PCI address.
type PciAddress C.struct_rte_pci_addr
