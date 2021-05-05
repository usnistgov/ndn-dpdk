package main

import (
	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/hrlog"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
)

const defaultStrategyName = "multicast"

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

	fib.GqlDefaultStrategy, e = strategycode.LoadFile(defaultStrategyName, "")
	if e != nil {
		return e
	}

	return nil
}
