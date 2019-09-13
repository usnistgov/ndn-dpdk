package fwdp

/*
#include "fwd.h"
#include "strategy.h"
*/
import "C"

import (
	"fmt"
	"time"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/container/strategycode"
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
)

type Fwd struct {
	dpdk.ThreadBase
	id int
	c  *C.FwFwd
}

func newFwd(id int) *Fwd {
	var fwd Fwd
	fwd.ResetThreadBase()
	fwd.id = id
	return &fwd
}

func (fwd *Fwd) String() string {
	return fmt.Sprintf("fwd%d", fwd.id)
}

func (fwd *Fwd) Init(fib *fib.Fib, pcctCfg pcct.Config, queueCap int, latencySampleFreq int) error {
	numaSocket := fwd.GetNumaSocket()

	queue, e := dpdk.NewRing(fwd.String()+"_queue", queueCap, numaSocket, false, true)
	if e != nil {
		return fmt.Errorf("dpdk.NewRing: %v", e)
	}

	pcctCfg.Id = fwd.String() + "_pcct"
	pcctCfg.NumaSocket = numaSocket
	pcct, e := pcct.New(pcctCfg)
	if e != nil {
		queue.Close()
		return fmt.Errorf("pcct.New: %v", e)
	}

	fwd.c = (*C.FwFwd)(dpdk.Zmalloc("FwFwd", C.sizeof_FwFwd, numaSocket))
	dpdk.InitStopFlag(unsafe.Pointer(&fwd.c.stop))
	fwd.c.id = C.uint8_t(fwd.id)
	fwd.c.queue = (*C.struct_rte_ring)(queue.GetPtr())

	fwd.c.fib = (*C.Fib)(fib.GetPtr(fwd.id))
	*C.FwFwd_GetPcctPtr_(fwd.c) = (*C.Pcct)(pcct.GetPtr())

	headerMp := appinit.MakePktmbufPool(appinit.MP_HDR, numaSocket)
	guiderMp := appinit.MakePktmbufPool(appinit.MP_INTG, numaSocket)
	indirectMp := appinit.MakePktmbufPool(appinit.MP_IND, numaSocket)

	fwd.c.headerMp = (*C.struct_rte_mempool)(headerMp.GetPtr())
	fwd.c.guiderMp = (*C.struct_rte_mempool)(guiderMp.GetPtr())
	fwd.c.indirectMp = (*C.struct_rte_mempool)(indirectMp.GetPtr())

	latencyStat := running_stat.FromPtr(unsafe.Pointer(&fwd.c.latencyStat))
	latencyStat.Clear(false)
	latencyStat.SetSampleRate(latencySampleFreq)

	fwd.c.suppressCfg.min = C.TscDuration(dpdk.ToTscDuration(10 * time.Millisecond))
	fwd.c.suppressCfg.multiplier = 2.0
	fwd.c.suppressCfg.max = C.TscDuration(dpdk.ToTscDuration(100 * time.Millisecond))

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
	return fwd.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&fwd.c.stop)))
}

func (fwd *Fwd) Close() error {
	queue := dpdk.RingFromPtr(unsafe.Pointer(fwd.c.queue))
	queue.Close()
	pcct := pcct.PcctFromPtr(unsafe.Pointer(*C.FwFwd_GetPcctPtr_(fwd.c)))
	pcct.Close()
	dpdk.Free(fwd.c)
	return nil
}

func init() {
	var nXsyms C.int
	strategycode.Xsyms = unsafe.Pointer(C.SgGetXsyms(&nXsyms))
	strategycode.NXsyms = int(nXsyms)
}
