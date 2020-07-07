package main

import (
	"flag"
	"os"

	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/core/yamlflag"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/createface"
)

type initConfig struct {
	Mempool    pktmbuf.TemplateUpdates
	LCoreAlloc ealthread.AllocConfig
	Face       createface.Config
	Ndt        ndt.Config
	Fib        fib.Config
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

func parseCommand(args []string) (initCfg initConfig, e error) {
	initCfg.Face.EnableEth = true
	initCfg.Ndt.PrefixLen = 2
	initCfg.Ndt.IndexBits = 16
	initCfg.Ndt.SampleFreq = 8
	initCfg.Fib.MaxEntries = 65535
	initCfg.Fib.NBuckets = 256
	initCfg.Fib.StartDepth = 8
	initCfg.Fwdp.FwdInterestQueue.DequeueBurstSize = 32
	initCfg.Fwdp.FwdDataQueue.DequeueBurstSize = 64
	initCfg.Fwdp.FwdNackQueue.DequeueBurstSize = 64
	initCfg.Fwdp.LatencySampleFreq = 16
	initCfg.Fwdp.PcctCapacity = 131071
	initCfg.Fwdp.CsCapMd = 32768
	initCfg.Fwdp.CsCapMi = 32768

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Var(yamlflag.New(&initCfg), "initcfg", "initialization config object")

	e = flags.Parse(args)
	if e != nil {
		return initConfig{}, e
	}

	return initCfg, nil
}
