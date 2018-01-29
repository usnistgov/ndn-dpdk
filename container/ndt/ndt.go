package ndt

/*
#include "ndt.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
)

type Config struct {
	PrefixLen  int
	IndexBits  int
	SampleFreq int
}

type Ndt struct {
	c *C.Ndt
}

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

func (ndt Ndt) Close() error {
	C.Ndt_Close(ndt.c)
	dpdk.Free(ndt.c)
	return nil
}

func (ndt Ndt) GetThread(i int) NdtThread {
	var cThreadPtr *C.NdtThread
	return NdtThread{ndt, *(**C.NdtThread)(unsafe.Pointer(uintptr(unsafe.Pointer(ndt.c.threads)) +
		uintptr(i)*uintptr(unsafe.Sizeof(cThreadPtr))))}
}

func (ndt Ndt) ReadCounters() (cnt []int) {
	cnt2 := make([]C.uint32_t, int(C.Ndt_SizeofCounters(ndt.c)))
	C.Ndt_ReadCounters(ndt.c, &cnt2[0])
	cnt = make([]int, len(cnt2))
	for i := 0; i < len(cnt); i++ {
		cnt[i] = int(cnt2[i])
	}
	return cnt
}

func (ndt Ndt) Update(hash uint64, value uint8) {
	C.Ndt_Update(ndt.c, C.uint64_t(hash), C.uint8_t(value))
}

type NdtThread struct {
	Ndt
	c *C.NdtThread
}

func (ndtt NdtThread) Lookup(name *ndn.Name) uint8 {
	return uint8(C.Ndt_Lookup(ndtt.Ndt.c, ndtt.c, (*C.Name)(name.GetPtr())))
}
