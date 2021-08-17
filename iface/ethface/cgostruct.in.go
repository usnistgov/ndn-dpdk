//go:build ignore

package ethface

/*
#include "../../csrc/ethface/locator.h"
*/
import "C"

type cLocator C.EthLocator

type cEtherAddr C.struct_rte_ether_addr
