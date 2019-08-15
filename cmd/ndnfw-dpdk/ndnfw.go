package main

import (
	"os"

	"ndn-dpdk/app/fwdp"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/dpdk"
)

var theDp *fwdp.DataPlane

func main() {
	initCfg, e := parseCommand(dpdk.MustInitEal(os.Args)[1:])
	if e != nil {
		log.WithError(e).Fatal("command line error")
	}
	log.WithField("nSlaves", len(dpdk.ListSlaveLCores())).Info("EAL ready")

	initCfg.InitConfig.Apply()

	startDp(initCfg.Ndt, initCfg.Fib, initCfg.Fwdp)
	startMgmt()

	select {}
}

func startDp(ndtCfg ndt.Config, fibCfg fib.Config, dpInit fwdpInitConfig) {
	var dpCfg fwdp.Config
	dpCfg.Ndt = ndtCfg
	dpCfg.Fib = fibCfg

	// set crypto config
	dpCfg.Crypto.InputCapacity = 64
	dpCfg.Crypto.OpPoolCapacity = 1023
	dpCfg.Crypto.OpPoolCacheSize = 31

	// set dataplane config
	dpCfg.FwdQueueCapacity = dpInit.FwdQueueCapacity
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
