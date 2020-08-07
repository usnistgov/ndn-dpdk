package main

import (
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/mgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/facemgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/fibmgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/fwdpmgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/hrlog"
	"github.com/usnistgov/ndn-dpdk/mgmt/ndtmgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/strategymgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/versionmgmt"
	"github.com/usnistgov/ndn-dpdk/strategy/strategyelf"
)

func startMgmt() {
	mgmt.Register(versionmgmt.VersionMgmt{})
	mgmt.Register(hrlog.HrlogMgmt{})

	mgmt.Register(facemgmt.FaceMgmt{})
	mgmt.Register(facemgmt.EthFaceMgmt{})

	mgmt.Register(ndtmgmt.NdtMgmt{
		Ndt: dp.GetNdt(),
	})

	mgmt.Register(strategymgmt.StrategyMgmt{})

	mgmt.Register(fibmgmt.FibMgmt{
		Fib:               dp.GetFib(),
		DefaultStrategyId: loadStrategy("multicast").ID(),
	})

	mgmt.Register(fwdpmgmt.DpInfoMgmt{Dp: dp})

	mgmt.Start()
}

func loadStrategy(shortname string) *strategycode.Strategy {
	logEntry := log.WithField("strategy", shortname)

	elf, e := strategyelf.Load(shortname)
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
