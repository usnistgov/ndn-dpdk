package pcct

/*
#include "pcct.h"
#include "pit.h"
#include "cs.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

type Config struct {
	Id         string
	MaxEntries int
	CsCapMd    int
	CsCapMi    int
	NumaSocket dpdk.NumaSocket
}

// The PIT-CS Composite Table (PCCT).
type Pcct struct {
	c *C.Pcct
}

// Create a PCCT, then initialize PIT and CS.
func New(cfg Config) (pcct *Pcct, e error) {
	idC := C.CString(cfg.Id)
	defer C.free(unsafe.Pointer(idC))
	pcct = new(Pcct)
	pcct.c = C.Pcct_New(idC, C.uint32_t(cfg.MaxEntries), C.unsigned(cfg.NumaSocket.ID()))
	if pcct.c == nil {
		return nil, dpdk.GetErrno()
	}

	pitC := C.Pit_FromPcct(pcct.c)
	C.Pit_Init(pitC)
	csC := C.Cs_FromPcct(pcct.c)
	C.Cs_Init(csC, C.uint32_t(cfg.CsCapMd), C.uint32_t(cfg.CsCapMi))
	return pcct, nil
}

func PcctFromPtr(ptr unsafe.Pointer) Pcct {
	return Pcct{(*C.Pcct)(ptr)}
}

// Get native *C.Pcct pointer to use in other packages.
func (pcct Pcct) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(pcct.c)
}

// Get underlying mempool of the PCCT.
func (pcct Pcct) GetMempool() dpdk.Mempool {
	return dpdk.MempoolFromPtr(pcct.GetPtr())
}

// Destroy the PCCT.
// Warning: currently this cannot release stored Interest/Data packets,
// and would cause memory leak.
func (pcct Pcct) Close() error {
	C.Pcct_Close(pcct.c)
	return nil
}
