package fwdp

/*
#include "../../csrc/fwdp/crypto.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/cryptodev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/multierr"
)

// CryptoConfig contains crypto helper thread configuration.
type CryptoConfig struct {
	InputCapacity  int `json:"inputCapacity,omitempty"`
	OpPoolCapacity int `json:"opPoolCapacity,omitempty"`
}

func (cfg *CryptoConfig) applyDefaults() {
	cfg.InputCapacity = ringbuffer.AlignCapacity(cfg.InputCapacity, iface.MaxBurstSize)
	if cfg.OpPoolCapacity <= 0 {
		cfg.OpPoolCapacity = 1023
	}
}

// Crypto represents a crypto helper thread.
type Crypto struct {
	ealthread.ThreadWithCtrl
	id int
	c  *C.FwCrypto
}

var (
	_ ealthread.ThreadWithRole     = (*Crypto)(nil)
	_ ealthread.ThreadWithLoadStat = (*Crypto)(nil)
)

// Init initializes the crypto helper thread.
func (fwc *Crypto) Init(lc eal.LCore, demuxPrep *demuxPreparer) error {
	socket := lc.NumaSocket()
	fwc.c = (*C.FwCrypto)(eal.ZmallocAligned("FwCrypto", C.sizeof_FwCrypto, 1, socket))
	fwc.ThreadWithCtrl = ealthread.NewThreadWithCtrl(
		cptr.Func0.C(unsafe.Pointer(C.FwCrypto_Run), unsafe.Pointer(fwc.c)),
		unsafe.Pointer(&fwc.c.ctrl),
	)
	fwc.SetLCore(lc)

	demuxPrep.PrepareDemuxD(iface.InputDemuxFromPtr(unsafe.Pointer(&fwc.c.output)))

	return nil
}

// Close stops and releases the thread.
func (fwc *Crypto) Close() error {
	fwc.Stop()
	eal.Free(fwc.c)
	return nil
}

func (fwc *Crypto) String() string {
	return fmt.Sprintf("crypto%d", fwc.id)
}

// ThreadRole implements ealthread.ThreadWithRole interface.
func (Crypto) ThreadRole() string {
	return RoleCrypto
}

func newCrypto(id int) *Crypto {
	return &Crypto{id: id}
}

// CryptoShared contains per NUMA socket shared resources for crypto helper threads.
type CryptoShared struct {
	input  *ringbuffer.Ring
	opPool *cryptodev.OpPool
	dev    *cryptodev.CryptoDev
}

// AssignTo assigns shared resources to crypto helper threads.
func (fwcsh *CryptoShared) AssignTo(fwcs []*Crypto) {
	qp := fwcsh.dev.QueuePairs()
	for i, fwc := range fwcs {
		fwc.c.input = (*C.struct_rte_ring)(fwcsh.input.Ptr())
		fwc.c.opPool = (*C.struct_rte_mempool)(fwcsh.opPool.Ptr())
		qp[i].CopyToC(unsafe.Pointer(&fwc.c.cqp))
	}
}

// ConnectTo connects forwarding thread to crypto input queue.
func (fwcsh *CryptoShared) ConnectTo(fwd *Fwd) {
	fwd.c.cryptoHelper = (*C.struct_rte_ring)(fwcsh.input.Ptr())
}

// Close deletes resources.
func (fwcsh *CryptoShared) Close() error {
	return multierr.Combine(
		fwcsh.dev.Close(),
		fwcsh.opPool.Close(),
		fwcsh.input.Close(),
	)
}

func newCryptoShared(cfg CryptoConfig, socket eal.NumaSocket, count int) (fwcsh *CryptoShared, e error) {
	cfg.applyDefaults()

	fwcsh = &CryptoShared{}

	ringConsumerMode := ringbuffer.ConsumerSingle
	if count > 1 {
		ringConsumerMode = ringbuffer.ConsumerMulti
	}
	fwcsh.input, e = ringbuffer.New(cfg.InputCapacity, socket, ringbuffer.ProducerMulti, ringConsumerMode)
	if e != nil {
		return nil, fmt.Errorf("ringbuffer.New: %w", e)
	}

	fwcsh.opPool, e = cryptodev.NewOpPool(cryptodev.OpPoolConfig{Capacity: cfg.OpPoolCapacity}, socket)
	if e != nil {
		return nil, fmt.Errorf("cryptodev.NewOpPool: %w", e)
	}

	var vcfg cryptodev.VDevConfig
	vcfg.Socket = socket
	vcfg.NQueuePairs = count
	fwcsh.dev, e = cryptodev.CreateVDev(vcfg)
	if e != nil {
		return nil, fmt.Errorf("cryptodev.CreateVDev: %w", e)
	}

	return fwcsh, nil
}
