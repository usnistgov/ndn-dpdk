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
	"go.uber.org/zap"
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
	logEntry := logger.With(zap.String("strategy", shortname))

	exeFile, e := os.Executable()
	if e != nil {
		logEntry.Fatal("os.Executable() error",
			zap.Error(e),
		)
	}
	elfFile := path.Join(path.Dir(exeFile), "../lib/bpf", "ndndpdk-strategy-"+shortname+".o")

	sc, e := strategycode.LoadFile(shortname, elfFile)
	if e != nil {
		logEntry.Fatal("strategycode.LoadFile() error",
			zap.String("filename", elfFile),
			zap.Error(e),
		)
	}

	logEntry.Debug("strategy loaded")
	return sc
}
