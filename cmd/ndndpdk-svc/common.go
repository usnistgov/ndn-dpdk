package main

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealinit"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"

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
