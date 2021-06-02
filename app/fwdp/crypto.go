package fwdp

/*
#include "../../csrc/fwdp/crypto.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/cryptodev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/mempool"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go4.org/must"
)

// CryptoConfig contains crypto helper thread configuration.
type CryptoConfig struct {
	InputCapacity  int `json:"inputCapacity,omitempty"`
	OpPoolCapacity int `json:"opPoolCapacity,omitempty"`
}

func (cfg *CryptoConfig) applyDefaults() {
	cfg.InputCapacity = ringbuffer.AlignCapacity(cfg.InputCapacity, 63)
	if cfg.OpPoolCapacity <= 0 {
		cfg.OpPoolCapacity = 1023
	}
}

// Crypto represents a crypto helper thread.
type Crypto struct {
	ealthread.Thread
	id     int
	c      *C.FwCrypto
	demuxD *iface.InputDemux
	devS   *cryptodev.CryptoDev
	devM   *cryptodev.CryptoDev
}

// Init initializes the crypto helper thread.
func (fwc *Crypto) Init(lc eal.LCore, cfg CryptoConfig, ndt *ndt.Ndt, fwds []*Fwd) error {
	cfg.applyDefaults()
	socket := lc.NumaSocket()
	fwc.c = (*C.FwCrypto)(eal.ZmallocAligned("FwCrypto", C.sizeof_FwCrypto, 1, socket))
	fwc.Thread = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.FwCrypto_Run), unsafe.Pointer(fwc.c)),
		ealthread.InitStopFlag(unsafe.Pointer(&fwc.c.stop)),
	)
	fwc.SetLCore(lc)

	input, e := ringbuffer.New(cfg.InputCapacity, socket,
		ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle)
	if e != nil {
		return fmt.Errorf("ringbuffer.New: %w", e)
	}
	fwc.c.input = (*C.struct_rte_ring)(input.Ptr())

	opPool, e := cryptodev.NewOpPool(cryptodev.OpPoolConfig{Capacity: cfg.OpPoolCapacity}, socket)
	if e != nil {
		return fmt.Errorf("cryptodev.NewOpPool: %w", e)
	}
	fwc.c.opPool = (*C.struct_rte_mempool)(opPool.Ptr())

	fwc.devS, e = cryptodev.SingleSegDrv.Create(cryptodev.Config{}, socket)
	if e != nil {
		return fmt.Errorf("cryptodev.SingleSegDrv.Create: %w", e)
	}
	fwc.devS.QueuePairs()[0].CopyToC(unsafe.Pointer(&fwc.c.singleSeg))

	fwc.devM, e = cryptodev.MultiSegDrv.Create(cryptodev.Config{}, socket)
	if e != nil {
		return fmt.Errorf("cryptodev.MultiSegDrv.Create: %w", e)
	}
	fwc.devM.QueuePairs()[0].CopyToC(unsafe.Pointer(&fwc.c.multiSeg))

	fwc.demuxD = iface.InputDemuxFromPtr(unsafe.Pointer(&fwc.c.output))
	fwc.demuxD.InitNdt(ndt.Threads()[fwc.id])
	for i, fwd := range fwds {
		fwc.demuxD.SetDest(i, fwd.queueD)
		fwd.c.crypto = fwc.c.input
	}

	return nil
}

// Close stops and releases the thread.
func (fwc *Crypto) Close() error {
	fwc.Stop()
	must.Close(fwc.devM)
	must.Close(fwc.devS)
	must.Close(mempool.FromPtr(unsafe.Pointer(fwc.c.opPool)))
	must.Close(ringbuffer.FromPtr(unsafe.Pointer(fwc.c.input)))
	eal.Free(unsafe.Pointer(fwc.c))
	return nil
}

func (fwc *Crypto) String() string {
	return fmt.Sprintf("crypto%d", fwc.id)
}

// ThreadRole implements ealthread.ThreadWithRole interface.
func (Crypto) ThreadRole() string {
	return RoleCrypto
}

// ThreadLoadStat implements ealthread.ThreadWithLoadStat interface.
func (fwc *Crypto) ThreadLoadStat() ealthread.LoadStat {
	return ealthread.LoadStatFromPtr(unsafe.Pointer(&fwc.c.loadStat))
}

func newCrypto(id int) *Crypto {
	return &Crypto{id: id}
}
