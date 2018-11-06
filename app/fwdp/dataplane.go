package fwdp

/*
#include "fwd.h"
#include "input.h"
#include "strategy.h"
*/
import "C"
import (
	"fmt"
	"time"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/container/strategycode"
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type Config struct {
	InputLCores []dpdk.LCore
	FwdLCores   []dpdk.LCore

	Ndt  ndt.Config  // NDT config
	Fib  fib.Config  // FIB config (Id ignored)
	Pcct pcct.Config // PCCT config template (Id and NumaSocket ignored)

	FwdQueueCapacity  int // input-fwd queue capacity, must be power of 2
	LatencySampleFreq int // latency sample frequency, between 0 and 30
}

// Forwarder data plane.
type DataPlane struct {
	ndt            *ndt.Ndt
	fib            *fib.Fib
	inputLCores    []dpdk.LCore
	inputs         []*C.FwInput
	inputRxLoopers []iface.IRxLooper
	fwdLCores      []dpdk.LCore
	fwds           []*C.FwFwd
}

func registerStrategyFuncs(vm unsafe.Pointer) error {
	if nErrors := C.SgRegisterFuncs((*C.struct_ubpf_vm)(vm)); nErrors > 0 {
		return fmt.Errorf("SgRegisterFuncs: %d errors", nErrors)
	}
	return nil
}

func New(cfg Config) (dp *DataPlane, e error) {
	nInputs := len(cfg.InputLCores)
	nFwds := len(cfg.FwdLCores)

	dp = new(DataPlane)
	dp.inputLCores = append([]dpdk.LCore{}, cfg.InputLCores...)
	dp.inputRxLoopers = make([]iface.IRxLooper, nInputs)
	dp.fwdLCores = append([]dpdk.LCore{}, cfg.FwdLCores...)

	dp.ndt = ndt.New(cfg.Ndt, dpdk.ListNumaSocketsOfLCores(dp.inputLCores))
	dp.ndt.Randomize(nFwds)

	cfg.Fib.Id = "FIB"
	if dp.fib, e = fib.New(cfg.Fib, dp.ndt, dpdk.ListNumaSocketsOfLCores(dp.fwdLCores)); e != nil {
		dp.Close()
		return nil, e
	}

	strategycode.RegisterStrategyFuncs = registerStrategyFuncs

	for i, lc := range cfg.FwdLCores {
		numaSocket := lc.GetNumaSocket()
		queue, e := dpdk.NewRing(fmt.Sprintf("FwFwdQ_%d", i), cfg.FwdQueueCapacity,
			numaSocket, false, true)
		if e != nil {
			dp.Close()
			return nil, fmt.Errorf("dpdk.NewRing(%d): %v", i, e)
		}

		pcctCfg := cfg.Pcct
		pcctCfg.Id = fmt.Sprintf("PCCT_%d", i)
		pcctCfg.NumaSocket = numaSocket
		pcct, e := pcct.New(pcctCfg)
		if e != nil {
			queue.Close()
			dp.Close()
			return nil, fmt.Errorf("pcct.New(%d): %v", i, e)
		}

		fwd := (*C.FwFwd)(dpdk.Zmalloc("FwFwd", C.sizeof_FwFwd, numaSocket))
		fwd.id = C.uint8_t(i)
		fwd.queue = (*C.struct_rte_ring)(queue.GetPtr())

		fwd.fib = (*C.Fib)(dp.fib.GetPtr(i))
		*C.__FwFwd_GetPcctPtr(fwd) = (*C.Pcct)(pcct.GetPtr())

		headerMp := appinit.MakePktmbufPool(appinit.MP_HDR, numaSocket)
		guiderMp := appinit.MakePktmbufPool(appinit.MP_INTG, numaSocket)
		indirectMp := appinit.MakePktmbufPool(appinit.MP_IND, numaSocket)

		fwd.headerMp = (*C.struct_rte_mempool)(headerMp.GetPtr())
		fwd.guiderMp = (*C.struct_rte_mempool)(guiderMp.GetPtr())
		fwd.indirectMp = (*C.struct_rte_mempool)(indirectMp.GetPtr())

		latencyStat := running_stat.FromPtr(unsafe.Pointer(&fwd.latencyStat))
		latencyStat.SetSampleRate(cfg.LatencySampleFreq)

		fwd.suppressCfg.min = C.TscDuration(dpdk.ToTscDuration(10 * time.Millisecond))
		fwd.suppressCfg.multiplier = 2.0
		fwd.suppressCfg.max = C.TscDuration(dpdk.ToTscDuration(100 * time.Millisecond))

		dp.fwds = append(dp.fwds, fwd)
	}

	for i, lc := range cfg.InputLCores {
		fwi := C.FwInput_New((*C.Ndt)(dp.ndt.GetPtr()), C.uint8_t(i),
			C.uint8_t(nFwds), C.unsigned(lc.GetNumaSocket()))
		if fwi == nil {
			dp.Close()
			return nil, dpdk.GetErrno()
		}

		for _, fwd := range dp.fwds {
			C.FwInput_Connect(fwi, fwd)
		}

		dp.inputs = append(dp.inputs, fwi)
	}

	return dp, nil
}

func (dp *DataPlane) Close() error {
	for _, fwi := range dp.inputs {
		dpdk.Free(fwi)
	}
	for _, fwd := range dp.fwds {
		queue := dpdk.RingFromPtr(unsafe.Pointer(fwd.queue))
		queue.Close()
		pcct := pcct.PcctFromPtr(unsafe.Pointer(*C.__FwFwd_GetPcctPtr(fwd)))
		pcct.Close()
		dpdk.Free(fwd)
	}
	if dp.fib != nil {
		dp.fib.Close()
	}
	if dp.ndt != nil {
		dp.ndt.Close()
	}
	return nil
}

// Provide crypto helper to fwds.
func (dp *DataPlane) SetCrypto(fwc *Crypto) {
	for _, fwd := range dp.fwds {
		fwd.crypto = fwc.c.input
	}
}

// Launch input process.
func (dp *DataPlane) LaunchInput(i int, rxl iface.IRxLooper, burstSize int) error {
	lc := dp.inputLCores[i]
	if state := lc.GetState(); state != dpdk.LCORE_STATE_WAIT {
		return fmt.Errorf("lcore %d for input %d is %s", lc, i, lc.GetState())
	}
	dp.inputRxLoopers[i] = rxl
	input := dp.inputs[i]

	ok := lc.RemoteLaunch(func() int {
		rxl.RxLoop(burstSize, unsafe.Pointer(C.FwInput_FaceRx), unsafe.Pointer(input))
		return 0
	})
	if !ok {
		return fmt.Errorf("failed to launch lcore %d for input %d", lc, i)
	}
	return nil
}

// Stop input process.
func (dp *DataPlane) StopInput(i int) {
	if rxl := dp.inputRxLoopers[i]; rxl == nil {
		return
	} else {
		rxl.StopRxLoop()
	}
	dp.inputRxLoopers[i] = nil
	dp.inputLCores[i].Wait()
}

// Launch forwarding process.
func (dp *DataPlane) LaunchFwd(i int) error {
	lc := dp.fwdLCores[i]
	if state := lc.GetState(); state != dpdk.LCORE_STATE_WAIT {
		return fmt.Errorf("lcore %d for fwd %d is %s", lc, i, lc.GetState())
	}
	fwd := dp.fwds[i]
	fwd.stop = C.bool(false)

	ok := lc.RemoteLaunch(func() int {
		rs := urcu.NewReadSide()
		defer rs.Close()
		C.FwFwd_Run(fwd)
		return 0
	})
	if !ok {
		return fmt.Errorf("failed to launch lcore %d for fwd %d", lc, i)
	}
	return nil
}

// Stop forwarding process.
func (dp *DataPlane) StopFwd(i int) {
	dp.fwds[i].stop = C.bool(true)
	dp.fwdLCores[i].Wait()
}
