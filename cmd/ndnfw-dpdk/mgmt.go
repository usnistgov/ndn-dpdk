package main

import (
	"time"

	"github.com/usnistgov/ndn-dpdk/appinit"
	"github.com/usnistgov/ndn-dpdk/container/ndt/ndtupdater"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/mgmt/facemgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/fibmgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/fwdpmgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/hrlog"
	"github.com/usnistgov/ndn-dpdk/mgmt/ndtmgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/strategymgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/versionmgmt"
	"github.com/usnistgov/ndn-dpdk/strategy/strategy_elf"
)

func startMgmt() {
	appinit.RegisterMgmt(versionmgmt.VersionMgmt{})
	appinit.RegisterMgmt(hrlog.HrlogMgmt{})

	appinit.RegisterMgmt(facemgmt.FaceMgmt{})
	appinit.RegisterMgmt(facemgmt.EthFaceMgmt{})

	appinit.RegisterMgmt(ndtmgmt.NdtMgmt{
		Ndt: theDp.GetNdt(),
		Updater: &ndtupdater.NdtUpdater{
			Ndt:      theDp.GetNdt(),
			Fib:      theDp.GetFib(),
			SleepFor: 200 * time.Millisecond,
		},
	})

	appinit.RegisterMgmt(strategymgmt.StrategyMgmt{})

	appinit.RegisterMgmt(fibmgmt.FibMgmt{
		Fib:               theDp.GetFib(),
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
