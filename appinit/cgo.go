package appinit

/*
#include <rte_config.h>
*/
import "C"

const (
	MEMPOOL_MAX_CACHE_SIZE = int(C.RTE_MEMPOOL_CACHE_MAX_SIZE)
)
