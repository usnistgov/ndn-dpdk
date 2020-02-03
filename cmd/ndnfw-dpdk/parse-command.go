package main

import (
	"flag"
	"os"

	"ndn-dpdk/app/fwdp"
	"ndn-dpdk/appinit"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/iface/createface"
)

type initConfig struct {
	appinit.InitConfig
	Ndt  ndt.Config
	Fib  fib.Config
	Fwdp fwdpInitConfig
}

type fwdpInitConfig struct {
	FwdInterestQueue  fwdp.FwdQueueConfig
	FwdDataQueue      fwdp.FwdQueueConfig
	FwdNackQueue      fwdp.FwdQueueConfig
	LatencySampleFreq int
	PcctCapacity      int
	CsCapMd           int
	CsCapMi           int
}

func parseCommand(args []string) (initCfg initConfig, e error) {
	initCfg.Face = createface.GetDefaultConfig()
	initCfg.Ndt.PrefixLen = 2
	initCfg.Ndt.IndexBits = 16
	initCfg.Ndt.SampleFreq = 8
	initCfg.Fib.MaxEntries = 65535
	initCfg.Fib.NBuckets = 256
	initCfg.Fib.StartDepth = 8
	initCfg.Fwdp.FwdInterestQueue.DequeueBurstSize = 48
	initCfg.Fwdp.FwdDataQueue.DequeueBurstSize = 64
	initCfg.Fwdp.FwdNackQueue.DequeueBurstSize = 64
	initCfg.Fwdp.LatencySampleFreq = 16
	initCfg.Fwdp.PcctCapacity = 131071
	initCfg.Fwdp.CsCapMd = 32768
	initCfg.Fwdp.CsCapMi = 32768

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	appinit.DeclareInitConfigFlag(flags, &initCfg)

	e = flags.Parse(args)
	if e != nil {
		return initConfig{}, e
	}

	return initCfg, nil
}
