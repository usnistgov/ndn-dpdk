package fwdp

/*
#include "../../csrc/fwdp/fwd.h"
#include "../../csrc/fwdp/strategy.h"
*/
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/container/pktqueue"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Forwarding thread.
type Fwd struct {
	ealthread.Thread
	id            int
	c             *C.FwFwd
	pcct          *pcct.Pcct
	interestQueue *pktqueue.PktQueue
	dataQueue     *pktqueue.PktQueue
	nackQueue     *pktqueue.PktQueue
}

func newFwd(id int) *Fwd {
	return &Fwd{
		id: id,
	}
}

func (fwd *Fwd) String() string {
	return fmt.Sprintf("fwd%d", fwd.id)
}

func (fwd *Fwd) Init(lc eal.LCore, fib *fib.Fib, pcctCfg pcct.Config, interestQueueCfg, dataQueueCfg, nackQueueCfg pktqueue.Config,
	latencySampleFreq int, suppressCfg pit.SuppressConfig) (e error) {
	socket := lc.NumaSocket()

	fwd.c = (*C.FwFwd)(eal.Zmalloc("FwFwd", C.sizeof_FwFwd, socket))
	fwd.c.id = C.uint8_t(fwd.id)
	fwd.Thread = ealthread.New(
		cptr.CFunction(unsafe.Pointer(C.FwFwd_Run), unsafe.Pointer(fwd.c)),
		ealthread.InitStopFlag(unsafe.Pointer(&fwd.c.stop)),
	)
	fwd.SetLCore(lc)

	if fwd.interestQueue, e = pktqueue.NewAt(unsafe.Pointer(&fwd.c.inInterestQueue), interestQueueCfg, socket); e != nil {
		return nil
	}
	if fwd.dataQueue, e = pktqueue.NewAt(unsafe.Pointer(&fwd.c.inDataQueue), dataQueueCfg, socket); e != nil {
		return nil
	}
	if fwd.nackQueue, e = pktqueue.NewAt(unsafe.Pointer(&fwd.c.inNackQueue), nackQueueCfg, socket); e != nil {
		return nil
	}

	fwd.c.fib = (*C.Fib)(fib.Ptr(fwd.id))

	pcctCfg.Socket = socket
	fwd.pcct, e = pcct.New(pcctCfg)
	if e != nil {
		return fmt.Errorf("pcct.New: %v", e)
	}
	*C.FwFwd_GetPcctPtr_(fwd.c) = (*C.Pcct)(fwd.pcct.Ptr())

	fwd.c.headerMp = (*C.struct_rte_mempool)(ndni.HeaderMempool.MakePool(socket).Ptr())
	fwd.c.guiderMp = (*C.struct_rte_mempool)(ndni.NameMempool.MakePool(socket).Ptr())
	fwd.c.indirectMp = (*C.struct_rte_mempool)(pktmbuf.Indirect.MakePool(socket).Ptr())

	latencyStat := runningstat.FromPtr(unsafe.Pointer(&fwd.c.latencyStat))
	latencyStat.Clear(false)
	latencyStat.SetSampleRate(latencySampleFreq)

	suppressCfg.CopyToC(unsafe.Pointer(&fwd.c.suppressCfg))

	return nil
}

func (fwd *Fwd) Close() error {
	fwd.Stop()
	fwd.interestQueue.Close()
	fwd.dataQueue.Close()
	fwd.nackQueue.Close()
	fwd.pcct.Close()
	eal.Free(fwd.c)
	return nil
}

func init() {
	var nXsyms C.int
	strategycode.Xsyms = unsafe.Pointer(C.SgGetXsyms(&nXsyms))
	strategycode.NXsyms = int(nXsyms)
}
