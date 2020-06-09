package fwdp

/*
#include "fwd.h"
#include "strategy.h"
*/
import "C"

import (
	"fmt"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/container/pit"
	"ndn-dpdk/container/pktqueue"
	"ndn-dpdk/container/strategycode"
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
)

// Forwarding thread.
type Fwd struct {
	dpdk.ThreadBase
	id            int
	c             *C.FwFwd
	interestQueue pktqueue.PktQueue
	dataQueue     pktqueue.PktQueue
	nackQueue     pktqueue.PktQueue
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
	numaSocket := fwd.GetNumaSocket()

	fwd.c = (*C.FwFwd)(dpdk.Zmalloc("FwFwd", C.sizeof_FwFwd, numaSocket))
	dpdk.InitStopFlag(unsafe.Pointer(&fwd.c.stop))
	fwd.c.id = C.uint8_t(fwd.id)

	if fwd.interestQueue, e = pktqueue.NewAt(unsafe.Pointer(&fwd.c.inInterestQueue), interestQueueCfg, fmt.Sprintf("%s_qI", fwd), numaSocket); e != nil {
		return nil
	}
	if fwd.dataQueue, e = pktqueue.NewAt(unsafe.Pointer(&fwd.c.inDataQueue), interestQueueCfg, fmt.Sprintf("%s_qD", fwd), numaSocket); e != nil {
		return nil
	}
	if fwd.nackQueue, e = pktqueue.NewAt(unsafe.Pointer(&fwd.c.inNackQueue), interestQueueCfg, fmt.Sprintf("%s_qN", fwd), numaSocket); e != nil {
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
	return fwd.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&fwd.c.stop)))
}

func (fwd *Fwd) Close() error {
	pktqueue.FromPtr(unsafe.Pointer(&fwd.c.inInterestQueue)).Close()
	pktqueue.FromPtr(unsafe.Pointer(&fwd.c.inDataQueue)).Close()
	pktqueue.FromPtr(unsafe.Pointer(&fwd.c.inNackQueue)).Close()
	pcct.PcctFromPtr(unsafe.Pointer(*C.FwFwd_GetPcctPtr_(fwd.c))).Close()
	dpdk.Free(fwd.c)
	return nil
}

func init() {
	var nXsyms C.int
	strategycode.Xsyms = unsafe.Pointer(C.SgGetXsyms(&nXsyms))
	strategycode.NXsyms = int(nXsyms)
}
