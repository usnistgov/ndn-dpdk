package iface

/*
#include "face.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

// Mempools for face construction.
type Mempools struct {
	IndirectMp dpdk.PktmbufPool
	NameMp     dpdk.PktmbufPool
	HeaderMp   dpdk.PktmbufPool
}

// Construct *C.FaceMempools.
func (mempools Mempools) GetPtr() unsafe.Pointer {
	var c C.FaceMempools
	c.indirectMp = (*C.struct_rte_mempool)(mempools.IndirectMp.GetPtr())
	c.nameMp = (*C.struct_rte_mempool)(mempools.NameMp.GetPtr())
	c.headerMp = (*C.struct_rte_mempool)(mempools.HeaderMp.GetPtr())
	return unsafe.Pointer(&c)
}
