package dpdk

/*
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
	c *C.struct_rte_mempool
}

func MempoolFromPtr(ptr unsafe.Pointer) Mempool {
	return Mempool{(*C.struct_rte_mempool)(ptr)}
}

// Get native *C.struct_rte_mempool pointer to use in other packages.
func (mp Mempool) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(mp.c)
}

func (mp Mempool) Close() error {
	C.rte_mempool_free(mp.c)
	return nil
}

func (mp Mempool) CountAvailable() int {
	return int(C.rte_mempool_avail_count(mp.c))
}

func (mp Mempool) CountInUse() int {
	return int(C.rte_mempool_in_use_count(mp.c))
}

type PktmbufPool struct {
	Mempool
}

func NewPktmbufPool(name string, capacity int, cacheSize int, privSize int,
	dataroomSize int, socket NumaSocket) (mp PktmbufPool, e error) {
	if privSize > C.UINT16_MAX {
		return mp, errors.New("privSize is too large")
	}
	if dataroomSize > C.UINT16_MAX {
		return mp, errors.New("dataroomSize is too large")
	}

	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	mp.c = C.rte_pktmbuf_pool_create(cName, C.uint(capacity), C.uint(cacheSize),
		C.uint16_t(privSize), C.uint16_t(dataroomSize), C.int(socket))
	if mp.c == nil {
		return mp, GetErrno()
	}
	return mp, nil
}

func (mp PktmbufPool) GetDataroom() int {
	return int(C.rte_pktmbuf_data_room_size(mp.c))
}

func (mp PktmbufPool) Alloc() (Mbuf, error) {
	m := C.rte_pktmbuf_alloc(mp.c)
	if m == nil {
		return Mbuf{}, errors.New("mbuf allocation failed")
	}
	return Mbuf{m}, nil
}

// Allocate several mbufs, writing into supplied slice of Mbuf or Packet.
func (mp PktmbufPool) AllocBulk(mbufs interface{}) error {
	ptr, count := ParseCptrArray(mbufs)
	res := C.rte_pktmbuf_alloc_bulk(mp.c, (**C.struct_rte_mbuf)(ptr), C.uint(count))
	if res != 0 {
		return errors.New("mbuf allocation failed")
	}
	return nil
}

// Clone a packet into indirect mbufs.
// Cloned segments point to the same memory and do not have copy-on-write semantics; appending new
// segments will not affect the original packet.
func (mp PktmbufPool) ClonePkt(pkt Packet) (Packet, error) {
	res := C.rte_pktmbuf_clone(pkt.c, mp.c)
	if res == nil {
		return Packet{}, errors.New("mbuf allocation failed")
	}
	return Packet{Mbuf{res}}, nil
}
