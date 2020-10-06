package main

import (
	"math/rand"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/hrlog"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealinit"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

var dp *fwdp.DataPlane

func main() {
	rand.Seed(time.Now().UnixNano())
	cfg := parseCommand()

	var req ealconfig.Request
	// main + CRYPTO + 2*FWD + (RX+TX)*(socket faces + Ethernet ports)
	req.MinLCores = 4 + 2*(1+len(cfg.Eal.PciDevices)+len(cfg.Eal.VirtualDevices))
	ealArgs, e := cfg.Eal.Args(req, nil)
	if e != nil {
		log.WithError(e).Fatal("EAL args error")
	}
	ealinit.Init(ealArgs)

	gqlserver.Start()
	hrlog.Init()

	cfg.Mempool.Apply()
	ealthread.DefaultAllocator.Config = cfg.LCoreAlloc

	startDp(cfg.Ndt, cfg.Fib, cfg.Fwdp)
	startMgmt()

	select {}
}

func startDp(ndtCfg ndt.Config, fibCfg fibdef.Config, dpInit fwdpInitConfig) {
	var dpCfg fwdp.Config
	dpCfg.Ndt = ndtCfg
	dpCfg.Fib = fibCfg
	dpCfg.Suppress = dpInit.Suppress

	// set crypto config
	dpCfg.Crypto.InputCapacity = 64
	dpCfg.Crypto.OpPoolCapacity = 1023

	// set dataplane config
	dpCfg.FwdInterestQueue = dpInit.FwdInterestQueue
	dpCfg.FwdDataQueue = dpInit.FwdDataQueue
	dpCfg.FwdNackQueue = dpInit.FwdNackQueue
	dpCfg.LatencySampleFreq = dpInit.LatencySampleFreq
	dpCfg.Pcct.MaxEntries = dpInit.PcctCapacity
	dpCfg.Pcct.CsCapMd = dpInit.CsCapMd
	dpCfg.Pcct.CsCapMi = dpInit.CsCapMi

	// create and launch dataplane
	var e error
	dp, e = fwdp.New(dpCfg)
	if e != nil {
		log.WithError(e).Fatal("dataplane init error")
	}

	log.Info("dataplane started")
}
