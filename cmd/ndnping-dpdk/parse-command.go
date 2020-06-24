package main

import (
	"flag"
	"os"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/ping"
	"github.com/usnistgov/ndn-dpdk/core/yamlflag"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface/createface"
)

type initConfig struct {
	Mempool    pktmbuf.TemplateUpdates
	LCoreAlloc eal.LCoreAllocConfig
	Face       createface.Config
}

type parsedCommand struct {
	initCfg         initConfig
	tasks           []ping.TaskConfig
	counterInterval time.Duration
}

func parseCommand(args []string) (pc parsedCommand, e error) {
	pc.initCfg.Face.EnableEth = true

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Var(yamlflag.New(&pc.initCfg), "initcfg", "initialization config object")
	flags.Var(yamlflag.New(&pc.tasks), "tasks", "ping task description")
	flags.DurationVar(&pc.counterInterval, "cnt", time.Second*10,
		"interval between printing counters (zero to disable)")

	e = flags.Parse(args)
	return pc, e
}
