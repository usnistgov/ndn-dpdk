//go:build ignore

package ethport

/*
#include "../../csrc/ethface/locator.h"
*/
import "C"

type LocatorC C.EthLocator

type EtherAddrC C.struct_rte_ether_addr

type Ip6AddrC C.struct_rte_ipv6_addr
