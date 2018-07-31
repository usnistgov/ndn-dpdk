package main

import (
	"time"

	"ndn-dpdk/appinit"
	"ndn-dpdk/container/ndt/ndtupdater"
	"ndn-dpdk/container/strategycode"
	"ndn-dpdk/iface/socketface"
	"ndn-dpdk/mgmt/facemgmt"
	"ndn-dpdk/mgmt/fibmgmt"
	"ndn-dpdk/mgmt/fwdpmgmt"
	"ndn-dpdk/mgmt/ndtmgmt"
	"ndn-dpdk/mgmt/strategymgmt"
	"ndn-dpdk/mgmt/versionmgmt"
	"ndn-dpdk/strategy/strategy_elf"
)

func startMgmt() {
	appinit.RegisterMgmt(versionmgmt.VersionMgmt{})

	if theSocketRxg != nil {
		facemgmt.CreateFace = socketface.MakeMgmtCreateFace(
			appinit.NewSocketFaceCfg(theSocketFaceNumaSocket), theSocketRxg, theSocketTxl,
			appinit.TheFaceQueueCapacityConfig.SocketTxPkts)
	}
	appinit.RegisterMgmt(facemgmt.FaceMgmt{})

	appinit.RegisterMgmt(ndtmgmt.NdtMgmt{
		Ndt: theNdt,
		Updater: &ndtupdater.NdtUpdater{
			Ndt:      theNdt,
			Fib:      theFib,
			SleepFor: 200 * time.Millisecond,
		},
	})

	appinit.RegisterMgmt(strategymgmt.StrategyMgmt{})

	appinit.RegisterMgmt(fibmgmt.FibMgmt{
		Fib:               theFib,
		DefaultStrategyId: loadStrategy("multicast").GetId(),
	})

	appinit.RegisterMgmt(fwdpmgmt.DpInfoMgmt{theDp})

	appinit.StartMgmt()
}

func loadStrategy(shortname string) strategycode.StrategyCode {
	logEntry := log.WithField("strategy", shortname)

	elf, e := strategy_elf.Load(shortname)
	if e != nil {
		logEntry.WithError(e).Fatal("strategy ELF load error")
	}
	sc, e := strategycode.Load(shortname, elf)
	if e != nil {
		logEntry.WithError(e).Fatal("strategy code load error")
	}

	logEntry.Debug("strategy loaded")
	return sc
}
