package ndt

/*
#include "../../csrc/ndt/ndt.h"
*/
import "C"
import (
	"math/rand"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Config contains NDT configuration.
type Config struct {
	PrefixLen  int
	IndexBits  int
	SampleFreq int
}

// Ndt represents a Name Dispatch Table (NDT).
type Ndt C.Ndt

// New creates an Ndt.
func New(cfg Config, sockets []eal.NumaSocket) (ndt *Ndt) {
	socketsC := make([]C.uint, len(sockets))
	for i, socket := range sockets {
		socketsC[i] = C.uint(socket.ID())
	}

	ndt = (*Ndt)(eal.Zmalloc("Ndt", C.sizeof_Ndt, sockets[0]))
	C.Ndt_Init(ndt.ptr(), C.uint16_t(cfg.PrefixLen), C.uint8_t(cfg.IndexBits), C.uint8_t(cfg.SampleFreq),
		C.uint8_t(len(sockets)), &socketsC[0])
	return ndt
}

func (ndt *Ndt) ptr() *C.Ndt {
	return (*C.Ndt)(ndt)
}

// Close destroys the NDT.
func (ndt *Ndt) Close() error {
	for i := 0; i < ndt.CountThreads(); i++ {
		eal.Free(ndt.getThreadC(i))
	}
	c := ndt.ptr()
	eal.Free(c.threads)
	eal.Free(c.table)
	eal.Free(c)
	return nil
}

// Ptr returns *C.Ndt pointer.
func (ndt *Ndt) Ptr() unsafe.Pointer {
	return unsafe.Pointer(ndt.ptr())
}

// CountElements returns number of table elements.
func (ndt *Ndt) CountElements() int {
	return int(ndt.ptr().indexMask + 1)
}

// GetPrefixLen returns number of name components used to compute hash.
func (ndt *Ndt) GetPrefixLen() int {
	return int(ndt.ptr().prefixLen)
}

// CountThreads returns number of threads.
func (ndt *Ndt) CountThreads() int {
	return int(ndt.ptr().nThreads)
}

func (ndt *Ndt) getThreadC(i int) *C.NdtThread {
	var threadPtrC *C.NdtThread
	first := uintptr(unsafe.Pointer(ndt.ptr().threads))
	offset := uintptr(i) * uintptr(unsafe.Sizeof(threadPtrC))
	return *(**C.NdtThread)(unsafe.Pointer(first + offset))
}

// GetThread returns a handle of i-th thread.
func (ndt *Ndt) GetThread(i int) *Thread {
	if i < 0 || i >= ndt.CountThreads() {
		return nil
	}
	return &Thread{ndt: ndt, c: ndt.getThreadC(i)}
}

// ComputeHash computes the hash used for a name.
func (ndt *Ndt) ComputeHash(name ndn.Name) uint64 {
	nameLen := len(name)
	if prefixLen := ndt.GetPrefixLen(); nameLen > prefixLen {
		nameLen = prefixLen
	}
	return ndni.CNameFromName(name).ComputePrefixHash(nameLen)
}

// GetIndex returns table index used for a hash.
func (ndt *Ndt) GetIndex(hash uint64) uint64 {
	return hash & uint64(ndt.ptr().indexMask)
}

// ReadElement reads a table element.
func (ndt *Ndt) ReadElement(index uint64) uint8 {
	return uint8(C.Ndt_ReadElement(ndt.ptr(), C.uint64_t(index)))
}

// ReadTable reads the entire table.
func (ndt *Ndt) ReadTable() (table []uint8) {
	table = make([]uint8, ndt.CountElements())
	for i := range table {
		table[i] = ndt.ReadElement(uint64(i))
	}
	return table
}

// ReadCounters reads all hit counters.
func (ndt *Ndt) ReadCounters() (cnt []int) {
	cnt = make([]int, ndt.CountElements())
	for i := 0; i < ndt.CountThreads(); i++ {
		threadC := ndt.getThreadC(i)
		first := uintptr(unsafe.Pointer(threadC)) + C.sizeof_NdtThread
		for j := range cnt {
			offset := uintptr(j) * C.sizeof_uint16_t
			cnt[j] += int(*(*C.uint16_t)(unsafe.Pointer(first + offset)))
		}
	}
	return cnt
}

// Update updates an element.
// When used with partitioned FIB, this function does not relocate FIB entries.
// See ndtupdater package for alternative.
func (ndt *Ndt) Update(index uint64, value uint8) {
	C.Ndt_Update(ndt.ptr(), C.uint64_t(index), C.uint8_t(value))
}

// Randomize updates all elements to random values < max.
// This should be used during initialization only.
func (ndt *Ndt) Randomize(max int) {
	for i, nElements := uint64(0), uint64(ndt.CountElements()); i < nElements; i++ {
		ndt.Update(i, uint8(rand.Intn(max)))
	}
}

// Lookup queries a name without incrementing hit counters.
func (ndt *Ndt) Lookup(name ndn.Name) (index uint64, value uint8) {
	cname := ndni.CNameFromName(name)
	pnameC := (*C.PName)(unsafe.Pointer(&cname.P))
	var indexC C.uint64_t
	value = uint8(C.Ndt_Lookup(ndt.ptr(), pnameC, (*C.uint8_t)(unsafe.Pointer(cname.V)), &indexC))
	return uint64(indexC), value
}

// Thread is a thread for NDT lookups.
type Thread struct {
	ndt *Ndt
	c   *C.NdtThread
}

// Lookup queries a name and increments hit counters.
func (ndtt *Thread) Lookup(name ndn.Name) uint8 {
	cname := ndni.CNameFromName(name)
	pnameC := (*C.PName)(unsafe.Pointer(&cname.P))
	return uint8(C.Ndtt_Lookup_(ndtt.ndt.ptr(), ndtt.c, pnameC, (*C.uint8_t)(unsafe.Pointer(cname.V))))
}
