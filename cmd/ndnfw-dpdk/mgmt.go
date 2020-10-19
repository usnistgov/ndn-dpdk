package main

import (
	"os"
	"path"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/mgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/facemgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/fibmgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/versionmgmt"
)

func startMgmt() {
	fwdp.GqlDataPlane = dp

	mgmt.Register(versionmgmt.VersionMgmt{})

	mgmt.Register(facemgmt.FaceMgmt{})

	ndt.GqlNdt = dp.Ndt()

	fib.GqlFib = dp.Fib()
	fib.GqlDefaultStrategy = loadStrategy("multicast")
	mgmt.Register(fibmgmt.FibMgmt{
		Fib:               fib.GqlFib,
		DefaultStrategyId: fib.GqlDefaultStrategy.ID(),
	})

	mgmt.Start()
}

func loadStrategy(shortname string) *strategycode.Strategy {
	logEntry := log.WithField("strategy", shortname)

	exeFile, e := os.Executable()
	if e != nil {
		logEntry.WithError(e).Fatal("os.Executable() error")
	}
	elfFile := path.Join(path.Dir(exeFile), "../lib/bpf", "ndndpdk-strategy-"+shortname+".o")

	sc, e := strategycode.LoadFile(shortname, elfFile)
	if e != nil {
		logEntry.WithField("filename", elfFile).WithError(e).Fatal("strategycode.LoadFile() error")
	}

	logEntry.Debug("strategy loaded")
	return sc
}
