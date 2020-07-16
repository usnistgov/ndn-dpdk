package pcct

/*
#include "../../csrc/pcct/pcct.h"
#include "../../csrc/pcct/pit.h"
#include "../../csrc/pcct/cs.h"
*/
import "C"
import (
	"errors"
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
func New(cfg Config) (pcct *Pcct, e error) {
	eltSize := C.sizeof_PccEntry
	if C.sizeof_PccEntryExt > eltSize {
		eltSize = C.sizeof_PccEntryExt
	}
	mp, e := mempool.New(mempool.Config{
		Capacity:       cfg.MaxEntries,
		ElementSize:    int(eltSize),
		PrivSize:       int(C.sizeof_Pcct),
		Socket:         cfg.Socket,
		NoCache:        true,
		SingleProducer: true,
		SingleConsumer: true,
	})
	if e != nil {
		return nil, e
	}

	mpC := (*C.struct_rte_mempool)(mp.Ptr())
	pcctC := (*C.Pcct)(C.rte_mempool_get_priv(mpC))
	*pcctC = C.Pcct{
		mp: mpC,
	}

	idC := C.CString(eal.AllocObjectID("pcct.tokenHt"))
	defer C.free(unsafe.Pointer(idC))
	if !bool(C.Pcct_Init(pcctC, idC, C.uint32_t(2*cfg.MaxEntries), C.uint(cfg.Socket.ID()))) {
		return nil, errors.New("Pcct_Init error")
	}

	C.Pit_Init(&pcctC.pit)
	C.Cs_Init(&pcctC.cs, C.uint32_t(cfg.CsCapMd), C.uint32_t(cfg.CsCapMi))
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
	return mempool.FromPtr(unsafe.Pointer(pcct.ptr().mp))
}

func (pcct *Pcct) String() string {
	return pcct.AsMempool().String()
}

// Close destroys the PCCT.
// This does not release stored Interest/Data packets.
func (pcct *Pcct) Close() error {
	C.Pcct_Clear(pcct.ptr())
	return pcct.AsMempool().Close()
}
