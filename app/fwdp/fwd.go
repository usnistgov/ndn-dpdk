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
	"github.com/usnistgov/ndn-dpdk/core/pcg32"
	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/multierr"
)

// Fwd represents a forwarding thread.
type Fwd struct {
	ealthread.ThreadWithCtrl
	id     int
	c      *C.FwFwd
	pcct   *pcct.Pcct
	queueI *iface.PktQueue
	queueD *iface.PktQueue
	queueN *iface.PktQueue
}

var (
	_ ealthread.ThreadWithRole     = (*Fwd)(nil)
	_ ealthread.ThreadWithLoadStat = (*Fwd)(nil)
)

// Init initializes the forwarding thread.
// Excluding FIB.
func (fwd *Fwd) Init(lc eal.LCore, pcctCfg pcct.Config, qcfgI, qcfgD, qcfgN iface.PktQueueConfig,
	latencySampleInterval int, suppressCfg pit.SuppressConfig) (e error) {
	socket := lc.NumaSocket()

	fwd.c = eal.Zmalloc[C.FwFwd]("FwFwd", C.sizeof_FwFwd, socket)
	fwd.c.id = C.uint8_t(fwd.id)
	fwd.ThreadWithCtrl = ealthread.NewThreadWithCtrl(
		cptr.Func0.C(C.FwFwd_Run, fwd.c),
		unsafe.Pointer(&fwd.c.ctrl),
	)
	fwd.SetLCore(lc)

	fwd.queueI = iface.PktQueueFromPtr(unsafe.Pointer(&fwd.c.queueI))
	if e = fwd.queueI.Init(qcfgI, socket); e != nil {
		return e
	}
	fwd.queueD = iface.PktQueueFromPtr(unsafe.Pointer(&fwd.c.queueD))
	if e = fwd.queueD.Init(qcfgD, socket); e != nil {
		return e
	}
	fwd.queueN = iface.PktQueueFromPtr(unsafe.Pointer(&fwd.c.queueN))
	if e = fwd.queueN.Init(qcfgN, socket); e != nil {
		return e
	}

	if fwd.pcct, e = pcct.New(pcctCfg, socket); e != nil {
		return fmt.Errorf("pcct.New: %w", e)
	}
	pcctC := (*C.Pcct)(fwd.pcct.Ptr())
	fwd.c.pit = &pcctC.pit
	fwd.c.cs = &pcctC.cs

	pcg32.Init(unsafe.Pointer(&fwd.c.sgRng))
	suppressCfg.CopyToC(unsafe.Pointer(&fwd.c.suppressCfg))
	(*ndni.Mempools)(unsafe.Pointer(&fwd.c.mp)).Assign(socket)
	fwd.LatencyStat().Init(latencySampleInterval)
	return nil
}

// Close stops and releases the thread.
func (fwd *Fwd) Close() error {
	defer eal.Free(fwd.c)
	return multierr.Combine(
		fwd.Stop(),
		fwd.queueI.Close(),
		fwd.queueD.Close(),
		fwd.queueN.Close(),
		fwd.pcct.Close(),
	)
}

// NumaSocket implements fib.LookupThread interface.
func (fwd *Fwd) NumaSocket() eal.NumaSocket {
	return fwd.LCore().NumaSocket()
}

// GetFibSgGlobal implements fib.LookupThread interface.
func (fwd *Fwd) GetFibSgGlobal() unsafe.Pointer {
	return unsafe.Pointer(&fwd.c.sgGlobal)
}

// GetFib implements fib.LookupThread interface.
func (fwd *Fwd) GetFib() (replica unsafe.Pointer, dynIndex int) {
	return unsafe.Pointer(fwd.c.fib), int(fwd.c.fibDynIndex)
}

// SetFib implements fib.LookupThread interface.
func (fwd *Fwd) SetFib(replica unsafe.Pointer, dynIndex int) {
	fwd.c.fib = (*C.Fib)(replica)
	fwd.c.fibDynIndex = C.uint8_t(dynIndex)
}

// Pit returns the PIT.
func (fwd *Fwd) Pit() *pit.Pit {
	return pit.FromPcct(fwd.pcct)
}

// Cs returns the CS.
func (fwd *Fwd) Cs() *cs.Cs {
	return cs.FromPcct(fwd.pcct)
}

// Counters retrieves forwarding thread counters.
func (fwd *Fwd) Counters() (cnt FwdCounters) {
	cnt.id = fwd.id
	cnt.NNoFibMatch = uint64(fwd.c.nNoFibMatch)
	cnt.NDupNonce = uint64(fwd.c.nDupNonce)
	cnt.NSgNoFwd = uint64(fwd.c.nSgNoFwd)
	cnt.NNackMismatch = uint64(fwd.c.nNackMismatch)
	return cnt
}

// LatencyStat returns latency statistics collector.
// Its reading reflects the latency since packet arrival until forwarding thread starts processing the packet.
func (fwd *Fwd) LatencyStat() *runningstat.RunningStat {
	return runningstat.FromPtr(unsafe.Pointer(&fwd.c.latencyStat))
}

// PktQueueOf returns PktQueue of specified PktType.
func (fwd *Fwd) PktQueueOf(t ndni.PktType) *iface.PktQueue {
	switch t {
	case ndni.PktInterest:
		return fwd.queueI
	case ndni.PktData:
		return fwd.queueD
	case ndni.PktNack:
		return fwd.queueN
	}
	return nil
}

func (fwd *Fwd) String() string {
	return fmt.Sprintf("fwd%d", fwd.id)
}

// ThreadRole implements ealthread.ThreadWithRole interface.
func (Fwd) ThreadRole() string {
	return RoleFwd
}

func newFwd(id int) *Fwd {
	return &Fwd{id: id}
}

// FwdCounters contains forwarding thread counters.
type FwdCounters struct {
	id            int    // FwFwd index
	NNoFibMatch   uint64 `json:"nNoFibMatch" gqldesc:"Interests dropped due to no FIB match."`
	NDupNonce     uint64 `json:"nDupNonce" gqldesc:"Interests dropped due to duplicate nonce."`
	NSgNoFwd      uint64 `json:"nSgNoFwd" gqldesc:"Interests not forwarded by strategy."`
	NNackMismatch uint64 `json:"nNackMismatch" gqldesc:"Nacks dropped due to outdated nonce."`
}

func init() {
	var nXsyms C.uint32_t
	xsyms := C.SgGetXsyms(&nXsyms)
	strategycode.XsymsMain.Assign(unsafe.Pointer(xsyms), int(nXsyms))
}
