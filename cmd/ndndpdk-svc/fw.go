package main

import (
	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/iface"
)

const defaultStrategyName = "multicast"

type fwArgs struct {
	CommonArgs
	fwdp.Config
}

func (a fwArgs) Activate() error {
	if e := a.CommonArgs.apply(); e != nil {
		return e
	}
	a.Config.LCoreAlloc = a.CommonArgs.LCoreAlloc

	dp, e := fwdp.New(a.Config)
	if e != nil {
		return e
	}

	fwdp.GqlDataPlane = dp
	iface.GqlCreateFaceAllowed = true
	ndt.GqlNdt = dp.Ndt()
	fib.GqlFib = dp.Fib()

	fib.GqlDefaultStrategy, e = strategycode.LoadFile(defaultStrategyName, "")
	if e != nil {
		return e
	}

	return nil
}
