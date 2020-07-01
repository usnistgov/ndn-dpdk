package pcct

/*
#include "../../csrc/pcct/pcct.h"
#include "../../csrc/pcct/pit.h"
#include "../../csrc/pcct/cs.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/mempool"
)

// Config contains PCCT configuration.
type Config struct {
	MaxEntries int
	CsCapMd    int
	CsCapMi    int
	Socket     eal.NumaSocket
}

// Pcct represents a PIT-CS Composite Table (PCCT).
type Pcct C.Pcct

// New creates a PCCT, and then initializes PIT and CS.
func New(id string, cfg Config) (pcct *Pcct, e error) {
	idC := C.CString(id)
	defer C.free(unsafe.Pointer(idC))
	pcctC := C.Pcct_New(idC, C.uint32_t(cfg.MaxEntries), C.uint(cfg.Socket.ID()))
	if pcctC == nil {
		return nil, eal.GetErrno()
	}

	pitC := C.Pit_FromPcct(pcctC)
	C.Pit_Init(pitC)
	csC := C.Cs_FromPcct(pcctC)
	C.Cs_Init(csC, C.uint32_t(cfg.CsCapMd), C.uint32_t(cfg.CsCapMi))
	return (*Pcct)(pcctC), nil
}

// Ptr returns *C.Pcct pointer.
func (pcct *Pcct) Ptr() unsafe.Pointer {
	return unsafe.Pointer(pcct)
}

func (pcct *Pcct) ptr() *C.Pcct {
	return (*C.Pcct)(pcct)
}

// AsMempool returns underlying mempool of the PCCT.
func (pcct *Pcct) AsMempool() *mempool.Mempool {
	return mempool.FromPtr(pcct.Ptr())
}

// Close destroys the PCCT.
// This does not release stored Interest/Data packets.
func (pcct *Pcct) Close() error {
	C.Pcct_Close(pcct.ptr())
	return nil
}
