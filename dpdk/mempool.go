package dpdk

/*
#include <rte_config.h>
#include <rte_mbuf.h>
#include <rte_mempool.h>
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"unsafe"
)

func computeMempoolCacheSize(capacity int) int {
	max := C.RTE_MEMPOOL_CACHE_MAX_SIZE
	if capacity/16 < max {
		return capacity / 16
	}
	min := max / 4
	for i := max; i >= min; i-- {
		if capacity%i == 0 {
			return i
		}
	}
	return max
}

type Mempool struct {
	c *C.struct_rte_mempool
}

func NewMempool(name string, capacity int, elementSize int, socket NumaSocket) (mp Mempool, e error) {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))

	mp.c = C.rte_mempool_create(nameC, C.uint(capacity), C.uint(elementSize),
		C.uint(computeMempoolCacheSize(capacity)), 0, nil, nil, nil, nil, C.int(socket), 0)
	if mp.c == nil {
		return mp, GetErrno()
	}
	return mp, nil
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

func (mp Mempool) SizeofElement() int {
	return int(mp.c.elt_size)
}

func (mp Mempool) CountAvailable() int {
	return int(C.rte_mempool_avail_count(mp.c))
}

func (mp Mempool) CountInUse() int {
	return int(C.rte_mempool_in_use_count(mp.c))
}

func (mp Mempool) Alloc() (ptr unsafe.Pointer) {
	res := C.rte_mempool_get(mp.c, &ptr)
	if res != 0 {
		return nil
	}
	return ptr
}

func (mp Mempool) AllocBulk(objs interface{}) error {
	ptr, count := ParseCptrArray(objs)
	if count == 0 {
		return nil
	}
	res := C.rte_mempool_get_bulk(mp.c, (*unsafe.Pointer)(ptr), C.uint(count))
	if res != 0 {
		return errors.New("mbuf allocation failed")
	}
	return nil
}

func (mp Mempool) Free(obj unsafe.Pointer) {
	C.rte_mempool_put(mp.c, obj)
}

func (mp Mempool) FreeBulk(objs interface{}) {
	ptr, count := ParseCptrArray(objs)
	if count == 0 {
		return
	}
	C.rte_mempool_put_bulk(mp.c, (*unsafe.Pointer)(ptr), C.uint(count))
}

type PktmbufPool struct {
	Mempool
}

func NewPktmbufPool(name string, capacity int, privSize int,
	dataroomSize int, socket NumaSocket) (mp PktmbufPool, e error) {
	if privSize > C.UINT16_MAX {
		return mp, errors.New("privSize is too large")
	}
	if dataroomSize > C.UINT16_MAX {
		return mp, errors.New("dataroomSize is too large")
	}

	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))

	mp.c = C.rte_pktmbuf_pool_create(nameC, C.uint(capacity), C.uint(computeMempoolCacheSize(capacity)),
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
