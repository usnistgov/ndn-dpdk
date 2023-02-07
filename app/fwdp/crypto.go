package fwdp

/*
#include "../../csrc/fwdp/crypto.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/cryptodev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
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
	_ DispatchThread               = (*Crypto)(nil)
)

// DispatchThreadID implements DispatchThread interface.
func (fwc *Crypto) DispatchThreadID() int {
	return fwc.id
}

func (fwc *Crypto) String() string {
	return fmt.Sprintf("crypto%d", fwc.id)
}

// DemuxOf implements DispatchThread interface.
func (fwc *Crypto) DemuxOf(t ndni.PktType) *iface.InputDemux {
	if t == ndni.PktData {
		return iface.InputDemuxFromPtr(unsafe.Pointer(&fwc.c.output))
	}
	return nil
}

// Close stops and releases the thread.
func (fwc *Crypto) Close() error {
	fwc.Stop()
	eal.Free(fwc.c)
	return nil
}

// ThreadRole implements ealthread.ThreadWithRole interface.
func (Crypto) ThreadRole() string {
	return RoleCrypto
}

// newCrypto creates a crypto helper thread.
func newCrypto(id int, lc eal.LCore, demuxPrep *demuxPreparer) (fwc *Crypto, e error) {
	socket := lc.NumaSocket()
	fwc = &Crypto{
		id: id,
		c:  eal.ZmallocAligned[C.FwCrypto]("FwCrypto", C.sizeof_FwCrypto, 1, socket),
	}
	fwc.ThreadWithCtrl = ealthread.NewThreadWithCtrl(
		cptr.Func0.C(C.FwCrypto_Run, unsafe.Pointer(fwc.c)),
		unsafe.Pointer(&fwc.c.ctrl),
	)
	fwc.SetLCore(lc)

	demuxPrep.Prepare(fwc, socket)
	return fwc, nil
}

// CryptoShared contains per NUMA socket shared resources for crypto helper threads.
type CryptoShared struct {
	input *ringbuffer.Ring
	dev   *cryptodev.CryptoDev
}

// AssignTo assigns shared resources to crypto helper threads.
func (fwcsh *CryptoShared) AssignTo(fwcs []*Crypto) {
	qp := fwcsh.dev.QueuePairs()
	for i, fwc := range fwcs {
		fwc.c.input = (*C.struct_rte_ring)(fwcsh.input.Ptr())
		qp[i].CopyToC(unsafe.Pointer(&fwc.c.cqp))
	}
}

// ConnectTo connects forwarding thread to crypto input queue.
func (fwcsh *CryptoShared) ConnectTo(fwd *Fwd) {
	fwd.c.cryptoHelper = (*C.struct_rte_ring)(fwcsh.input.Ptr())
}

// Close deletes resources.
func (fwcsh *CryptoShared) Close() error {
	return errors.Join(
		fwcsh.dev.Close(),
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

	var vcfg cryptodev.VDevConfig
	vcfg.Socket = socket
	vcfg.NQueuePairs = count
	fwcsh.dev, e = cryptodev.CreateVDev(vcfg)
	if e != nil {
		return nil, fmt.Errorf("cryptodev.CreateVDev: %w", e)
	}

	return fwcsh, nil
}
