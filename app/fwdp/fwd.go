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
	"ndn-dpdk/container/codel_queue"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/container/strategycode"
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
)

type FwdQueueConfig struct {
	codel_queue.Config
	Capacity int // queue capacity, must be power of 2, default 131072
}

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

func (fwd *Fwd) Init(fib *fib.Fib, pcctCfg pcct.Config, interestQueueCfg, dataQueueCfg, nackQueueCfg FwdQueueConfig, latencySampleFreq int) error {
	numaSocket := fwd.GetNumaSocket()

	fwd.c = (*C.FwFwd)(dpdk.Zmalloc("FwFwd", C.sizeof_FwFwd, numaSocket))
	dpdk.InitStopFlag(unsafe.Pointer(&fwd.c.stop))
	fwd.c.id = C.uint8_t(fwd.id)

	if e := fwd.initQueue("_qI", interestQueueCfg, &fwd.c.inInterestQueue); e != nil {
		return nil
	}
	if e := fwd.initQueue("_qD", dataQueueCfg, &fwd.c.inDataQueue); e != nil {
		return nil
	}
	if e := fwd.initQueue("_qN", nackQueueCfg, &fwd.c.inNackQueue); e != nil {
		return nil
	}

	fwd.c.fib = (*C.Fib)(fib.GetPtr(fwd.id))

	pcctCfg.Id = fwd.String() + "_pcct"
	pcctCfg.NumaSocket = numaSocket
	if pcct, e := pcct.New(pcctCfg); e != nil {
		return fmt.Errorf("pcct.New: %v", e)
	} else {
		*C.FwFwd_GetPcctPtr_(fwd.c) = (*C.Pcct)(pcct.GetPtr())
	}

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

func (fwd *Fwd) initQueue(suffix string, cfg FwdQueueConfig, q *C.CoDelQueue) error {
	capacity := cfg.Capacity
	if capacity == 0 {
		capacity = 131072
	}

	ring, e := dpdk.NewRing(fmt.Sprintf("%s_%s", fwd, suffix), capacity, fwd.GetNumaSocket(), false, true)
	if e != nil {
		return fmt.Errorf("dpdk.NewRing: %v", e)
	}

	codel_queue.NewAt(unsafe.Pointer(q), cfg.Config, ring)
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
	codel_queue.FromPtr(unsafe.Pointer(&fwd.c.inInterestQueue)).GetRing().Close()
	codel_queue.FromPtr(unsafe.Pointer(&fwd.c.inDataQueue)).GetRing().Close()
	codel_queue.FromPtr(unsafe.Pointer(&fwd.c.inNackQueue)).GetRing().Close()
	pcct.PcctFromPtr(unsafe.Pointer(*C.FwFwd_GetPcctPtr_(fwd.c))).Close()
	dpdk.Free(fwd.c)
	return nil
}

func init() {
	var nXsyms C.int
	strategycode.Xsyms = unsafe.Pointer(C.SgGetXsyms(&nXsyms))
	strategycode.NXsyms = int(nXsyms)
}
