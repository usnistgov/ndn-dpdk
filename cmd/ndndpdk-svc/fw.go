package main

import (
	"os"
	"path"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/hrlog"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
)

type fwArgs struct {
	CommonArgs
	fwdp.Config
}

func (a fwArgs) Activate() error {
	var req ealconfig.Request
	// main + CRYPTO + 2*FWD + (RX+TX)*(socket faces + Ethernet ports)
	req.MinLCores = 4 + 2*(1+len(a.Eal.PciDevices)+len(a.Eal.VirtualDevices))
	if e := a.CommonArgs.apply(req); e != nil {
		return e
	}
	hrlog.Init()

	dp, e := fwdp.New(a.Config)
	if e != nil {
		return e
	}

	fwdp.GqlDataPlane = dp
	ndt.GqlNdt = dp.Ndt()
	fib.GqlFib = dp.Fib()
	fib.GqlDefaultStrategy = loadStrategy("multicast")
	return nil
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
