package pcct

/*
#include "pcct.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

type Config struct {
	Id         string
	MaxEntries int
	NumaSocket dpdk.NumaSocket
}

type Pcct struct {
	c *C.Pcct
}

func New(cfg Config) (pcct *Pcct, e error) {
	idC := C.CString(cfg.Id)
	defer C.free(unsafe.Pointer(idC))
	pcct = new(Pcct)
	pcct.c = C.Pcct_New(idC, C.uint32_t(cfg.MaxEntries), C.unsigned(cfg.NumaSocket))
	if pcct.c == nil {
		return nil, dpdk.GetErrno()
	}
	return pcct, nil
}

func (pcct Pcct) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(pcct.c)
}

// Get underlying mempool of the PCCT.
func (pcct Pcct) GetMempool() dpdk.Mempool {
	return dpdk.MempoolFromPtr(pcct.GetPtr())
}

func (pcct *Pcct) Close() error {
	if pcct.c == nil {
		return nil
	}
	C.Pcct_Close(pcct.c)
	pcct.c = nil
	return nil
}
