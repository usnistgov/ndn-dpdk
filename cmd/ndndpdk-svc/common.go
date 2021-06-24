package main

import (
	"time"

	"github.com/usnistgov/ndn-dpdk/bpf"
	"github.com/usnistgov/ndn-dpdk/container/hrlog"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealinit"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethvdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/zap"

	_ "github.com/usnistgov/ndn-dpdk/iface/ethface"
	_ "github.com/usnistgov/ndn-dpdk/iface/socketface"
)

// CommonArgs contains arguments shared between forwarder and traffic generator.
type CommonArgs struct {
	Eal     ealconfig.Config        `json:"eal,omitempty"`
	Mempool pktmbuf.TemplateUpdates `json:"mempool,omitempty"`
	Hrlog   bool                    `json:"hrlog,omitempty"`
}

func (a CommonArgs) apply() error {
	ealArgs, e := a.Eal.Args(nil)
	if e != nil {
		return e
	}
	ealinit.Init(ealArgs)
	a.Mempool.Apply()
	if a.Hrlog {
		hrlog.Init()
	}
	return nil
}

func initXDPProgram() {
	path, e := bpf.XDP.Find("map0")
	if e != nil {
		logger.Warn("XDP program not found, AF_XDP may not work correctly", zap.Error(e))
		return
	}

	ethvdev.XDPProgram = path
}

func delayedShutdown(then func()) {
	go func() {
		iface.CloseAll()

		time.Sleep(100 * time.Millisecond)
		then()
	}()
}
