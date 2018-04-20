package ndt

/*
#include "ndt.h"
*/
import "C"
import (
	"math/rand"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
)

type Config struct {
	PrefixLen  int
	IndexBits  int
	SampleFreq int
}

// The Name Dispatch Table (NDT).
type Ndt struct {
	c *C.Ndt
}

// Create an NDT.
func New(cfg Config, numaSockets []dpdk.NumaSocket) (ndt Ndt) {
	numaSocketsC := make([]C.unsigned, len(numaSockets))
	for i, socket := range numaSockets {
		numaSocketsC[i] = C.unsigned(socket)
	}

	ndt.c = (*C.Ndt)(dpdk.Zmalloc("Ndt", C.sizeof_Ndt, numaSockets[0]))
	C.Ndt_Init(ndt.c, C.uint16_t(cfg.PrefixLen), C.uint8_t(cfg.IndexBits), C.uint8_t(cfg.SampleFreq),
		C.uint8_t(len(numaSockets)), &numaSocketsC[0])
	return ndt
}

// Destroy the NDT.
func (ndt Ndt) Close() error {
	C.Ndt_Close(ndt.c)
	dpdk.Free(ndt.c)
	return nil
}

// Get native *C.Ndt pointer to use in other packages.
func (ndt Ndt) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(ndt.c)
}

// Get number of table elements.
func (ndt Ndt) CountElements() int {
	return int(ndt.c.indexMask + 1)
}

func (ndt Ndt) getThreadC(i int) *C.NdtThread {
	var threadPtrC *C.NdtThread
	first := uintptr(unsafe.Pointer(ndt.c.threads))
	offset := uintptr(i) * uintptr(unsafe.Sizeof(threadPtrC))
	return *(**C.NdtThread)(unsafe.Pointer(first + offset))
}

// Obtain a handle of NdtThread for lookups.
func (ndt Ndt) GetThread(i int) NdtThread {
	return NdtThread{ndt, ndt.getThreadC(i)}
}

// Read the table.
func (ndt Ndt) ReadTable() (table []uint8) {
	table = make([]uint8, ndt.CountElements())
	C.rte_memcpy(unsafe.Pointer(&table[0]), unsafe.Pointer(ndt.c.table), C.size_t(len(table)))
	return table
}

// Read hit counters.
func (ndt Ndt) ReadCounters() (cnt []int) {
	cnt = make([]int, ndt.CountElements())
	for i := C.uint8_t(0); i < ndt.c.nThreads; i++ {
		threadC := ndt.getThreadC(int(i))
		first := uintptr(unsafe.Pointer(threadC)) + C.sizeof_NdtThread
		for i := range cnt {
			offset := uintptr(i) * C.sizeof_uint16_t
			cnt[i] += int(*(*C.uint16_t)(unsafe.Pointer(first + offset)))
		}
	}
	return cnt
}

// Update an element.
func (ndt Ndt) Update(hash uint64, value uint8) {
	C.Ndt_Update(ndt.c, C.uint64_t(hash), C.uint8_t(value))
}

// Update all elements to random values < max.
// This should be used during initialization only.
func (ndt Ndt) Randomize(max int) {
	for i, nElements := uint64(0), uint64(ndt.CountElements()); i < nElements; i++ {
		ndt.Update(i, uint8(rand.Intn(max)))
	}
}

// A thread for NDT lookups.
type NdtThread struct {
	Ndt
	c *C.NdtThread
}

// Lookup a name.
func (ndtt NdtThread) Lookup(name *ndn.Name) uint8 {
	return uint8(C.Ndt_Lookup(ndtt.Ndt.c, ndtt.c, (*C.PName)(name.GetPNamePtr()),
		(*C.uint8_t)(name.GetValue().GetPtr())))
}
