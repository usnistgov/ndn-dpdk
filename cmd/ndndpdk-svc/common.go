package main

import (
	"github.com/usnistgov/ndn-dpdk/bpf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealinit"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethvdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"go.uber.org/zap"

	_ "github.com/usnistgov/ndn-dpdk/iface/ethface"
	_ "github.com/usnistgov/ndn-dpdk/iface/socketface"
)

// CommonArgs contains arguments shared between forwarder and traffic generator.
type CommonArgs struct {
	Eal        ealconfig.Config        `json:"eal,omitempty"`
	Mempool    pktmbuf.TemplateUpdates `json:"mempool,omitempty"`
	LCoreAlloc ealthread.AllocConfig   `json:"lcoreAlloc,omitempty"`
}

func (a CommonArgs) apply(req ealconfig.Request) error {
	ealArgs, e := a.Eal.Args(req, nil)
	if e != nil {
		return e
	}
	ealinit.Init(ealArgs)
	a.Mempool.Apply()
	ealthread.DefaultAllocator.Config = a.LCoreAlloc
	return nil
}

func initXDPProgram() {
	path, e := bpf.XDP.Find("map0")
	if e != nil {
		logger.Warn("XDP program not found, AF_XDP may not work correctly",
			zap.Error(e),
		)
		return
	}

	ethvdev.XDPProgram = path
}
