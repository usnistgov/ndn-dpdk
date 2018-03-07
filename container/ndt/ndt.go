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
	return int(C.Ndt_CountElements(ndt.c))
}

// Obtain a handle of NdtThread for lookups.
func (ndt Ndt) GetThread(i int) NdtThread {
	var cThreadPtr *C.NdtThread
	return NdtThread{ndt, *(**C.NdtThread)(unsafe.Pointer(uintptr(unsafe.Pointer(ndt.c.threads)) +
		uintptr(i)*uintptr(unsafe.Sizeof(cThreadPtr))))}
}

// Read hit counters.
func (ndt Ndt) ReadCounters() (cnt []int) {
	cnt2 := make([]C.uint32_t, ndt.CountElements())
	C.Ndt_ReadCounters(ndt.c, &cnt2[0])
	cnt = make([]int, len(cnt2))
	for i, c := range cnt2 {
		cnt[i] = int(c)
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

// Perform a non-thread-safe lookup.
func (ndtt NdtThread) Lookup(name *ndn.Name) uint8 {
	return uint8(C.Ndt_Lookup(ndtt.Ndt.c, ndtt.c, (*C.PName)(name.GetPNamePtr()),
		(*C.uint8_t)(name.GetValue().GetPtr())))
}
