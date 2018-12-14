package main

import (
	"flag"
	"os"

	"ndn-dpdk/appinit"
	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/createface"
)

type initConfig struct {
	appinit.InitConfig `yaml:",inline"`
	Ndt                ndt.Config
	Fib                fib.Config
	Fwdp               fwdpInitConfig
}

type fwdpInitConfig struct {
	InputLCores  []dpdk.LCore
	CryptoLCores []dpdk.LCore
	FwdLCores    []dpdk.LCore

	FwdQueueCapacity  int
	LatencySampleFreq int
	PcctCapacity      int
	CsCapMd           int
	CsCapMi           int

	AutoFaces bool
}

func parseCommand(args []string) (initCfg initConfig, e error) {
	initCfg.Face = createface.GetDefaultConfig()
	initCfg.Ndt.PrefixLen = 2
	initCfg.Ndt.IndexBits = 16
	initCfg.Ndt.SampleFreq = 8
	initCfg.Fib.MaxEntries = 65535
	initCfg.Fib.NBuckets = 256
	initCfg.Fib.StartDepth = 8
	initCfg.Fwdp.FwdQueueCapacity = 128
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
