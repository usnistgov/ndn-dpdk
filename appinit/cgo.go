package appinit

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk

#include <rte_config.h>
*/
import "C"

const (
	MEMPOOL_MAX_CACHE_SIZE = int(C.RTE_MEMPOOL_CACHE_MAX_SIZE)
)
