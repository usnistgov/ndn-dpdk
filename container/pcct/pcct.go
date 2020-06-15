package pcct

/*
#include "../../csrc/pcct/pcct.h"
#include "../../csrc/pcct/pit.h"
#include "../../csrc/pcct/cs.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk/eal"
	"ndn-dpdk/dpdk/mempool"
)

type Config struct {
	Id         string
	MaxEntries int
	CsCapMd    int
	CsCapMi    int
	NumaSocket eal.NumaSocket
}

// The PIT-CS Composite Table (PCCT).
type Pcct C.Pcct

// Create a PCCT, then initialize PIT and CS.
func New(cfg Config) (pcct *Pcct, e error) {
	idC := C.CString(cfg.Id)
	defer C.free(unsafe.Pointer(idC))
	pcctC := C.Pcct_New(idC, C.uint32_t(cfg.MaxEntries), C.uint(cfg.NumaSocket.ID()))
	if pcctC == nil {
		return nil, eal.GetErrno()
	}

	pitC := C.Pit_FromPcct(pcctC)
	C.Pit_Init(pitC)
	csC := C.Cs_FromPcct(pcctC)
	C.Cs_Init(csC, C.uint32_t(cfg.CsCapMd), C.uint32_t(cfg.CsCapMi))
	return (*Pcct)(pcctC), nil
}

func PcctFromPtr(ptr unsafe.Pointer) *Pcct {
	return (*Pcct)(ptr)
}

// Get native *C.Pcct pointer to use in other packages.
func (pcct *Pcct) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(pcct)
}

func (pcct *Pcct) getPtr() *C.Pcct {
	return (*C.Pcct)(pcct)
}

// Get underlying mempool of the PCCT.
func (pcct *Pcct) GetMempool() *mempool.Mempool {
	return mempool.FromPtr(pcct.GetPtr())
}

// Destroy the PCCT.
// Warning: currently this cannot release stored Interest/Data packets,
// and would cause memory leak.
func (pcct *Pcct) Close() error {
	C.Pcct_Close(pcct.getPtr())
	return nil
}
