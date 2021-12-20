//go:build ignore

package ethport

/*
#include "../../csrc/ethface/locator.h"
*/
import "C"

type CLocator C.EthLocator

type CEtherAddr C.struct_rte_ether_addr
