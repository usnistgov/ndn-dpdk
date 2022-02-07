// Package pcct implements the PIT-CS Composite Table.
package pcct

/*
#include "../../csrc/pcct/pcct.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/mempool"
	"go.uber.org/zap"
)

var logger = logging.New("pcct")

// PIT and CS initialization functions.
// These are assigned during package pit and package cs initialization.
var (
	InitPit func(cfg Config, pcct *Pcct)
	InitCs  func(cfg Config, pcct *Pcct)
)

// Config contains PCCT configuration.
type Config struct {
	PcctCapacity       int `json:"pcctCapacity,omitempty"`
	CsMemoryCapacity   int `json:"csMemoryCapacity,omitempty"`
	CsDiskCapacity     int `json:"csDiskCapacity,omitempty"`
	CsIndirectCapacity int `json:"csIndirectCapacity,omitempty"`
}

func (cfg *Config) applyDefaults() {
	if cfg.PcctCapacity <= 0 {
		cfg.PcctCapacity = 131071
	}
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
		return nil, fmt.Errorf("mempool.New error: %w", e)
	}

	mpC := (*C.struct_rte_mempool)(mp.Ptr())
	pcctC := (*C.Pcct)(C.rte_mempool_get_priv(mpC))
	*pcctC = C.Pcct{
		mp: mpC,
	}

	tokenHtID := C.CString(eal.AllocObjectID("pcct.tokenHt"))
	defer C.free(unsafe.Pointer(tokenHtID))
	if ok := bool(C.Pcct_Init(pcctC, tokenHtID, C.uint32_t(cfg.PcctCapacity), C.int(socket.ID()))); !ok {
		return nil, fmt.Errorf("Pcct_Init error: %w", eal.GetErrno())
	}

	pcct = (*Pcct)(pcctC)
	logger.Info("init",
		zap.Uintptr("pcct", uintptr(unsafe.Pointer(pcct))),
		zap.Uintptr("mp", uintptr(unsafe.Pointer(mpC))),
		zap.Uintptr("token-ht", uintptr(unsafe.Pointer(pcct.tokenHt))),
	)

	InitPit(cfg, pcct)
	InitCs(cfg, pcct)
	return pcct, nil
}
