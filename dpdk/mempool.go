package dpdk

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk

#include <rte_config.h>
#include <rte_mbuf.h>
#include <rte_mempool.h>
#include <stdlib.h> // free()
*/
import "C"
import (
	"errors"
	"unsafe"
)

type Mempool struct {
	ptr *C.struct_rte_mempool
}

func (mp Mempool) Close() {
	C.rte_mempool_free(mp.ptr)
}

type PktmbufPool struct {
	Mempool
}

func NewPktmbufPool(name string, capacity uint, cacheSize uint, privSize uint16,
	dataRoomSize uint16, socket NumaSocket) (PktmbufPool, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var mp PktmbufPool
	mp.ptr = C.rte_pktmbuf_pool_create(cName, C.uint(capacity), C.uint(cacheSize),
		C.uint16_t(privSize), C.uint16_t(dataRoomSize), C.int(socket))
	if mp.ptr == nil {
		return mp, GetErrno()
	}
	return mp, nil
}

func (mp PktmbufPool) Alloc() (Mbuf, error) {
	m := C.rte_pktmbuf_alloc(mp.ptr)
	if m == nil {
		return Mbuf{nil}, errors.New("mbuf allocation failed")
	}
	return Mbuf{m}, nil
}
