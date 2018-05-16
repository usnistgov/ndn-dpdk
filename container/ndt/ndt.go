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
func New(cfg Config, numaSockets []dpdk.NumaSocket) (ndt *Ndt) {
	ndt = new(Ndt)

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
func (ndt *Ndt) Close() error {
	for i := 0; i < ndt.CountThreads(); i++ {
		dpdk.Free(ndt.getThreadC(i))
	}
	dpdk.Free(ndt.c.threads)
	dpdk.Free(ndt.c.table)
	dpdk.Free(ndt.c)
	return nil
}

// Get native *C.Ndt pointer to use in other packages.
func (ndt *Ndt) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(ndt.c)
}

// Get number of table elements.
func (ndt *Ndt) CountElements() int {
	return int(ndt.c.indexMask + 1)
}

// Get number of name components used to compute hash.
func (ndt *Ndt) GetPrefixLen() int {
	return int(ndt.c.prefixLen)
}

// Get number of threads.
func (ndt *Ndt) CountThreads() int {
	return int(ndt.c.nThreads)
}

func (ndt *Ndt) getThreadC(i int) *C.NdtThread {
	var threadPtrC *C.NdtThread
	first := uintptr(unsafe.Pointer(ndt.c.threads))
	offset := uintptr(i) * uintptr(unsafe.Sizeof(threadPtrC))
	return *(**C.NdtThread)(unsafe.Pointer(first + offset))
}

func (ndt *Ndt) GetThread(i int) NdtThread {
	return NdtThread{ndt: ndt, c: ndt.getThreadC(i)}
}

// Compute the hash used for a name.
func (ndt *Ndt) ComputeHash(name *ndn.Name) uint64 {
	prefixLen := name.Len()
	if prefixLen > int(ndt.c.prefixLen) {
		prefixLen = int(ndt.c.prefixLen)
	}
	return name.ComputePrefixHash(prefixLen)
}

// Get table index used for a hash.
func (ndt *Ndt) GetIndex(hash uint64) uint64 {
	return hash & uint64(ndt.c.indexMask)
}

// Read a table element.
func (ndt *Ndt) ReadElement(index uint64) uint8 {
	return uint8(C.Ndt_ReadElement(ndt.c, C.uint64_t(index)))
}

// Read the table.
func (ndt *Ndt) ReadTable() (table []uint8) {
	table = make([]uint8, ndt.CountElements())
	for i := range table {
		table[i] = ndt.ReadElement(uint64(i))
	}
	return table
}

// Read hit counters.
func (ndt *Ndt) ReadCounters() (cnt []int) {
	cnt = make([]int, ndt.CountElements())
	for i := 0; i < ndt.CountThreads(); i++ {
		threadC := ndt.getThreadC(i)
		first := uintptr(unsafe.Pointer(threadC)) + C.sizeof_NdtThread
		for i := range cnt {
			offset := uintptr(i) * C.sizeof_uint16_t
			cnt[i] += int(*(*C.uint16_t)(unsafe.Pointer(first + offset)))
		}
	}
	return cnt
}

// Update an element.
// When used with partitioned FIB, this function does not relocate FIB entries.
// See ndtupdater package for alternative.
func (ndt *Ndt) Update(index uint64, value uint8) {
	C.Ndt_Update(ndt.c, C.uint64_t(index), C.uint8_t(value))
}

// Update all elements to random values < max.
// This should be used during initialization only.
func (ndt *Ndt) Randomize(max int) {
	for i, nElements := uint64(0), uint64(ndt.CountElements()); i < nElements; i++ {
		ndt.Update(i, uint8(rand.Intn(max)))
	}
}

// Lookup a name without counting.
func (ndt *Ndt) Lookup(name *ndn.Name) (index uint64, value uint8) {
	var indexC C.uint64_t
	value = uint8(C.Ndt_Lookup(ndt.c, (*C.PName)(name.GetPNamePtr()),
		(*C.uint8_t)(name.GetValue().GetPtr()), &indexC))
	return uint64(indexC), value
}

// A thread for NDT lookups.
type NdtThread struct {
	ndt *Ndt
	c   *C.NdtThread
}

// Lookup a name with counting.
func (ndtt *NdtThread) Lookup(name *ndn.Name) uint8 {
	return uint8(C.__Ndtt_Lookup(ndtt.ndt.c, ndtt.c, (*C.PName)(name.GetPNamePtr()),
		(*C.uint8_t)(name.GetValue().GetPtr())))
}
