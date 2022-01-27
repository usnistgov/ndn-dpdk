// Package pcct implements the PIT-CS Composite Table.
package pcct

/*
#include "../../csrc/pcct/pcct.h"
#include "../../csrc/pcct/pit.h"
#include "../../csrc/pcct/cs.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/mempool"
)

// Config contains PCCT configuration.
type Config struct {
	PcctCapacity       int `json:"pcctCapacity,omitempty"`
	CsDirectCapacity   int `json:"csDirectCapacity,omitempty"`
	CsIndirectCapacity int `json:"csIndirectCapacity,omitempty"`
}

func (cfg *Config) applyDefaults() {
	if cfg.PcctCapacity <= 0 {
		cfg.PcctCapacity = 131071
	}

	adjustCsCap := func(v *int) {
		if *v <= 0 {
			*v = cfg.PcctCapacity / 4
		}
		// use C.CsEvictBulk instead of cs.EvictBulk to avoid circular dependency
		*v = math.MaxInt(*v, C.CsEvictBulk)
	}
	adjustCsCap(&cfg.CsDirectCapacity)
	adjustCsCap(&cfg.CsIndirectCapacity)
}

// Pcct represents a PIT-CS Composite Table (PCCT).
type Pcct C.Pcct

// Ptr returns *C.Pcct pointer.
func (pcct *Pcct) Ptr() unsafe.Pointer {
	return unsafe.Pointer(pcct)
}

// AsMempool returns underlying mempool of the PCCT.
func (pcct *Pcct) AsMempool() *mempool.Mempool {
	return mempool.FromPtr(unsafe.Pointer(pcct.mp))
}

func (pcct *Pcct) String() string {
	return pcct.AsMempool().String()
}

// Close destroys the PCCT.
func (pcct *Pcct) Close() error {
	C.Pcct_Clear((*C.Pcct)(pcct))
	return pcct.AsMempool().Close()
}

// New creates a PCCT and initializes PIT and CS.
func New(cfg Config, socket eal.NumaSocket) (pcct *Pcct, e error) {
	cfg.applyDefaults()
	mp, e := mempool.New(mempool.Config{
		Capacity:       cfg.PcctCapacity,
		ElementSize:    math.MaxInt(int(C.sizeof_PccEntry), int(C.sizeof_PccEntryExt)),
		PrivSize:       int(C.sizeof_Pcct),
		Socket:         socket,
		NoCache:        true,
		SingleProducer: true,
		SingleConsumer: true,
	})
	if e != nil {
		return nil, fmt.Errorf("mempool.New error %w", e)
	}

	mpC := (*C.struct_rte_mempool)(mp.Ptr())
	pcctC := (*C.Pcct)(C.rte_mempool_get_priv(mpC))
	*pcctC = C.Pcct{
		mp: mpC,
	}

	tokenHtID := C.CString(eal.AllocObjectID("pcct.tokenHt"))
	defer C.free(unsafe.Pointer(tokenHtID))
	if ok := bool(C.Pcct_Init(pcctC, tokenHtID, C.uint32_t(cfg.PcctCapacity), C.int(socket.ID()))); !ok {
		return nil, fmt.Errorf("Pcct_Init error %w", eal.GetErrno())
	}

	C.Pit_Init(&pcctC.pit)
	C.Cs_Init(&pcctC.cs, C.uint32_t(cfg.CsDirectCapacity), C.uint32_t(cfg.CsIndirectCapacity))
	return (*Pcct)(pcctC), nil
}
