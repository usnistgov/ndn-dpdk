package main

import (
	"os"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/mgmt/hrlog"
)

var theDp *fwdp.DataPlane

func main() {
	initCfg, e := parseCommand(eal.InitEal(os.Args)[1:])
	if e != nil {
		log.WithError(e).Fatal("command line error")
	}
	log.WithField("nSlaves", len(eal.ListSlaveLCores())).Info("EAL ready")
	hrlog.Init()

	initCfg.InitConfig.Apply()

	startDp(initCfg.Ndt, initCfg.Fib, initCfg.Fwdp)
	startMgmt()

	select {}
}

func startDp(ndtCfg ndt.Config, fibCfg fib.Config, dpInit fwdpInitConfig) {
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

	// create dataplane
	{
		var e error
		theDp, e = fwdp.New(dpCfg)
		if e != nil {
			log.WithError(e).Fatal("dataplane init error")
		}
	}

	// launch dataplane
	if e := theDp.Launch(); e != nil {
		log.WithError(e).Fatal("dataplane launch error")
	}
	log.Info("dataplane started")
}
