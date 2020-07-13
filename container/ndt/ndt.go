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
// sockets indicates the NUMA sockets of lookup threads.
// The table is allocated on sockets[0].
func New(cfg Config, sockets []eal.NumaSocket) *Ndt {
	tableSize := uintptr(1) << cfg.IndexBits
	threadSize := uintptr(C.sizeof_NdtThread) + tableSize*uintptr(C.sizeof_uint16_t)

	c := (*C.Ndt)(eal.Zmalloc("Ndt", C.sizeof_Ndt, sockets[0]))
	c.indexMask = C.uint64_t(tableSize - 1)
	c.sampleMask = C.uint64_t(1)<<cfg.SampleFreq - 1
	c.prefixLen = C.uint16_t(cfg.PrefixLen)
	c.nThreads = C.uint8_t(len(sockets))

	c.table = (*C.uint8_t)(eal.Zmalloc("NdtTable", tableSize, sockets[0]))
	for i, socket := range sockets {
		c.threads[i] = (*C.NdtThread)(eal.Zmalloc("NdtThread", threadSize, socket))
	}
	return (*Ndt)(c)
}

// Ptr returns *C.Ndt pointer.
func (ndt *Ndt) Ptr() unsafe.Pointer {
	return unsafe.Pointer(ndt.ptr())
}

func (ndt *Ndt) ptr() *C.Ndt {
	return (*C.Ndt)(ndt)
}

// Close destroys the NDT.
func (ndt *Ndt) Close() error {
	c := ndt.ptr()
	for i, end := 0, int(c.nThreads); i < end; i++ {
		eal.Free(c.threads[i])
	}
	eal.Free(c.table)
	eal.Free(c)
	return nil
}

// CountElements returns number of table elements.
func (ndt *Ndt) CountElements() int {
	return int(ndt.ptr().indexMask + 1)
}

// PrefixLen returns number of name components used to compute hash.
func (ndt *Ndt) PrefixLen() int {
	return int(ndt.ptr().prefixLen)
}

// Threads returns lookup threads.
func (ndt *Ndt) Threads() (list []*Thread) {
	c := ndt.ptr()
	list = make([]*Thread, int(c.nThreads))
	for i := range list {
		list[i] = &Thread{ndt: ndt, c: c.threads[i]}
	}
	return list
}

// ComputeHash computes the hash used for a name.
func (ndt *Ndt) ComputeHash(name ndn.Name) uint64 {
	if prefixLen := ndt.PrefixLen(); len(name) > prefixLen {
		name = name[:prefixLen]
	}
	pname := ndni.NewPName(name)
	defer pname.Free()
	return pname.ComputeHash()
}

// IndexOfHash returns table index used for a hash.
func (ndt *Ndt) IndexOfHash(hash uint64) uint64 {
	return hash & uint64(ndt.ptr().indexMask)
}

// IndexOfName returns table index used for a name.
func (ndt *Ndt) IndexOfName(name ndn.Name) uint64 {
	return ndt.IndexOfHash(ndt.ComputeHash(name))
}

// Read reads a table element.
func (ndt *Ndt) Read(index uint64) uint8 {
	return uint8(C.Ndt_Read(ndt.ptr(), C.uint64_t(index)))
}

// ReadTable reads the entire table.
func (ndt *Ndt) ReadTable() (table []uint8) {
	table = make([]uint8, ndt.CountElements())
	for i := range table {
		table[i] = ndt.Read(uint64(i))
	}
	return table
}

// ReadCounters reads all hit counters.
func (ndt *Ndt) ReadCounters() (cnt []int) {
	cnt = make([]int, ndt.CountElements())
	for _, ndtt := range ndt.Threads() {
		for index := range cnt {
			cnt[index] += int(ndtt.readCounter(uint64(index)))
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
	for i, end := uint64(0), uint64(ndt.CountElements()); i < end; i++ {
		ndt.Update(i, uint8(rand.Intn(max)))
	}
}

// Lookup queries a name without incrementing hit counters.
func (ndt *Ndt) Lookup(name ndn.Name) (index uint64, value uint8) {
	nameP := ndni.NewPName(name)
	defer nameP.Free()
	var indexC C.uint64_t
	value = uint8(C.Ndt_Lookup(ndt.ptr(), (*C.PName)(nameP.Ptr()), &indexC))
	return uint64(indexC), value
}

// Thread is a thread for NDT lookups.
type Thread struct {
	ndt *Ndt
	c   *C.NdtThread
}

// Lookup queries a name and increments hit counters.
func (ndtt *Thread) Lookup(name ndn.Name) uint8 {
	nameP := ndni.NewPName(name)
	defer nameP.Free()
	return uint8(C.Ndtt_Lookup(ndtt.ndt.ptr(), ndtt.c, (*C.PName)(nameP.Ptr())))
}

func (ndtt *Thread) readCounter(index uint64) uint16 {
	offset := C.sizeof_NdtThread + uintptr(index)*C.sizeof_uint16_t
	return *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(ndtt.c)) + offset))
}
