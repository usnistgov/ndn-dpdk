package fwdp

/*
#include "fwd.h"
#include "strategy.h"
*/
import "C"

import (
	"fmt"
	"unsafe"

	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/container/pit"
	"ndn-dpdk/container/pktqueue"
	"ndn-dpdk/container/strategycode"
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk/eal"
	"ndn-dpdk/dpdk/pktmbuf"
	"ndn-dpdk/ndn"
)

// Forwarding thread.
type Fwd struct {
	eal.ThreadBase
	id            int
	c             *C.FwFwd
	pcct          *pcct.Pcct
	interestQueue *pktqueue.PktQueue
	dataQueue     *pktqueue.PktQueue
	nackQueue     *pktqueue.PktQueue
}

func newFwd(id int) *Fwd {
	var fwd Fwd
	fwd.id = id
	return &fwd
}

func (fwd *Fwd) String() string {
	return fmt.Sprintf("fwd%d", fwd.id)
}

func (fwd *Fwd) Init(fib *fib.Fib, pcctCfg pcct.Config, interestQueueCfg, dataQueueCfg, nackQueueCfg pktqueue.Config,
	latencySampleFreq int, suppressCfg pit.SuppressConfig) (e error) {
	socket := fwd.GetNumaSocket()

	fwd.c = (*C.FwFwd)(eal.Zmalloc("FwFwd", C.sizeof_FwFwd, socket))
	eal.InitStopFlag(unsafe.Pointer(&fwd.c.stop))
	fwd.c.id = C.uint8_t(fwd.id)

	if fwd.interestQueue, e = pktqueue.NewAt(unsafe.Pointer(&fwd.c.inInterestQueue), interestQueueCfg, fmt.Sprintf("%s_qI", fwd), socket); e != nil {
		return nil
	}
	if fwd.dataQueue, e = pktqueue.NewAt(unsafe.Pointer(&fwd.c.inDataQueue), dataQueueCfg, fmt.Sprintf("%s_qD", fwd), socket); e != nil {
		return nil
	}
	if fwd.nackQueue, e = pktqueue.NewAt(unsafe.Pointer(&fwd.c.inNackQueue), nackQueueCfg, fmt.Sprintf("%s_qN", fwd), socket); e != nil {
		return nil
	}

	fwd.c.fib = (*C.Fib)(fib.GetPtr(fwd.id))

	pcctCfg.Id = fwd.String() + "_pcct"
	pcctCfg.NumaSocket = socket
	fwd.pcct, e = pcct.New(pcctCfg)
	if e != nil {
		return fmt.Errorf("pcct.New: %v", e)
	}
	*C.FwFwd_GetPcctPtr_(fwd.c) = (*C.Pcct)(fwd.pcct.GetPtr())

	fwd.c.headerMp = (*C.struct_rte_mempool)(ndn.HeaderMempool.MakePool(socket).GetPtr())
	fwd.c.guiderMp = (*C.struct_rte_mempool)(ndn.NameMempool.MakePool(socket).GetPtr())
	fwd.c.indirectMp = (*C.struct_rte_mempool)(pktmbuf.Indirect.MakePool(socket).GetPtr())

	latencyStat := running_stat.FromPtr(unsafe.Pointer(&fwd.c.latencyStat))
	latencyStat.Clear(false)
	latencyStat.SetSampleRate(latencySampleFreq)

	suppressCfg.CopyToC(unsafe.Pointer(&fwd.c.suppressCfg))

	return nil
}

func (fwd *Fwd) Launch() error {
	return fwd.LaunchImpl(func() int {
		rs := urcu.NewReadSide()
		defer rs.Close()
		C.FwFwd_Run(fwd.c)
		return 0
	})
}

func (fwd *Fwd) Stop() error {
	return fwd.StopImpl(eal.NewStopFlag(unsafe.Pointer(&fwd.c.stop)))
}

func (fwd *Fwd) Close() error {
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
