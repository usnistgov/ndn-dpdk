package main

import (
	"flag"

	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/core/yamlflag"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"

	_ "github.com/usnistgov/ndn-dpdk/iface/ethface"
	_ "github.com/usnistgov/ndn-dpdk/iface/socketface"
)

type initConfig struct {
	Eal        ealconfig.Config `json:"eal"`
	Mempool    pktmbuf.TemplateUpdates
	LCoreAlloc ealthread.AllocConfig
	Ndt        ndt.Config
	Fib        fibdef.Config
	Fwdp       fwdpInitConfig
}

type fwdpInitConfig struct {
	FwdInterestQueue  iface.PktQueueConfig
	FwdDataQueue      iface.PktQueueConfig
	FwdNackQueue      iface.PktQueueConfig
	LatencySampleFreq int
	Suppress          pit.SuppressConfig
	PcctCapacity      int
	CsCapMd           int
	CsCapMi           int
}

func parseCommand() (cfg initConfig) {
	cfg.Fwdp.FwdInterestQueue.DequeueBurstSize = 32
	cfg.Fwdp.FwdDataQueue.DequeueBurstSize = 64
	cfg.Fwdp.FwdNackQueue.DequeueBurstSize = 64
	cfg.Fwdp.LatencySampleFreq = 16
	cfg.Fwdp.PcctCapacity = 131071
	cfg.Fwdp.CsCapMd = 32768
	cfg.Fwdp.CsCapMi = 32768

	flag.Var(yamlflag.New(&cfg), "initcfg", "initialization config object")
	flag.Parse()
	return cfg
}
