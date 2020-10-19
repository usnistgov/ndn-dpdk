package fwdp

/*
#include "../../csrc/fwdp/fwd.h"
#include "../../csrc/fwdp/strategy.h"
*/
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/cs"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Fwd represents a forwarding thread.
type Fwd struct {
	ealthread.Thread
	id     int
	c      *C.FwFwd
	pcct   *pcct.Pcct
	queueI *iface.PktQueue
	queueD *iface.PktQueue
	queueN *iface.PktQueue
}

func newFwd(id int) *Fwd {
	return &Fwd{
		id: id,
	}
}

func (fwd *Fwd) String() string {
	return fmt.Sprintf("fwd%d", fwd.id)
}

// Init initializes the forwarding thread.
// Excluding FIB.
func (fwd *Fwd) Init(lc eal.LCore, pcctCfg pcct.Config, qcfgI, qcfgD, qcfgN iface.PktQueueConfig,
	latencySampleFreq int, suppressCfg pit.SuppressConfig) (e error) {
	socket := lc.NumaSocket()

	fwd.c = (*C.FwFwd)(eal.Zmalloc("FwFwd", C.sizeof_FwFwd, socket))
	fwd.c.id = C.uint8_t(fwd.id)
	fwd.Thread = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.FwFwd_Run), unsafe.Pointer(fwd.c)),
		ealthread.InitStopFlag(unsafe.Pointer(&fwd.c.stop)),
	)
	fwd.SetLCore(lc)

	fwd.queueI = iface.PktQueueFromPtr(unsafe.Pointer(&fwd.c.queueI))
	if e := fwd.queueI.Init(qcfgI, socket); e != nil {
		return e
	}
	fwd.queueD = iface.PktQueueFromPtr(unsafe.Pointer(&fwd.c.queueD))
	if e := fwd.queueD.Init(qcfgD, socket); e != nil {
		return e
	}
	fwd.queueN = iface.PktQueueFromPtr(unsafe.Pointer(&fwd.c.queueN))
	if e := fwd.queueN.Init(qcfgN, socket); e != nil {
		return e
	}

	fwd.pcct, e = pcct.New(pcctCfg, socket)
	if e != nil {
		return fmt.Errorf("pcct.New: %w", e)
	}
	pcctC := (*C.Pcct)(fwd.pcct.Ptr())
	fwd.c.pit = &pcctC.pit
	fwd.c.cs = &pcctC.cs

	fwd.c.headerMp = (*C.struct_rte_mempool)(ndni.HeaderMempool.MakePool(socket).Ptr())
	fwd.c.indirectMp = (*C.struct_rte_mempool)(pktmbuf.Indirect.MakePool(socket).Ptr())

	latencyStat := runningstat.FromPtr(unsafe.Pointer(&fwd.c.latencyStat))
	latencyStat.Clear(false)
	latencyStat.SetSampleRate(latencySampleFreq)

	suppressCfg.CopyToC(unsafe.Pointer(&fwd.c.suppressCfg))

	return nil
}

// Close stops and releases the forwarding thread.
func (fwd *Fwd) Close() error {
	fwd.Stop()
	fwd.queueI.Close()
	fwd.queueD.Close()
	fwd.queueN.Close()
	fwd.pcct.Close()
	eal.Free(fwd.c)
	return nil
}

// NumaSocket implements fib.LookupThread.
func (fwd *Fwd) NumaSocket() eal.NumaSocket {
	return fwd.Thread.LCore().NumaSocket()
}

// SetFib implements fib.LookupThread.
func (fwd *Fwd) SetFib(replica unsafe.Pointer, index int) {
	fwd.c.fib = (*C.Fib)(replica)
	fwd.c.fibDynIndex = C.uint8_t(index)
}

// Pit returns the PIT.
func (fwd *Fwd) Pit() *pit.Pit {
	return pit.FromPcct(fwd.pcct)
}

// Cs returns the CS.
func (fwd *Fwd) Cs() *cs.Cs {
	return cs.FromPcct(fwd.pcct)
}

// FwdCounters contains forwarding thread counters.
type FwdCounters struct {
	id            int    // FwFwd index
	NNoFibMatch   uint64 `json:"nNoFibMatch"`   // Interests dropped due to no FIB match
	NDupNonce     uint64 `json:"nDupNonce"`     // Interests dropped due to duplicate nonce
	NSgNoFwd      uint64 `json:"nSgNoFwd"`      // Interests not forwarded by strategy
	NNackMismatch uint64 `json:"nNackMismatch"` // Nacks dropped due to outdated nonce
}

// ReadCounters retrieves forwarding thread counters.
func (fwd *Fwd) ReadCounters() (cnt FwdCounters) {
	cnt.id = fwd.id
	cnt.NNoFibMatch = uint64(fwd.c.nNoFibMatch)
	cnt.NDupNonce = uint64(fwd.c.nDupNonce)
	cnt.NSgNoFwd = uint64(fwd.c.nSgNoFwd)
	cnt.NNackMismatch = uint64(fwd.c.nNackMismatch)
	return cnt
}

func init() {
	var nXsyms C.int
	strategycode.Xsyms = unsafe.Pointer(C.SgGetXsyms(&nXsyms))
	strategycode.NXsyms = int(nXsyms)
}
