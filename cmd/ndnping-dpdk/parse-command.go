package main

import (
	"flag"
	"os"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/ping"
	"github.com/usnistgov/ndn-dpdk/core/yamlflag"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"

	_ "github.com/usnistgov/ndn-dpdk/iface/ethface"
	_ "github.com/usnistgov/ndn-dpdk/iface/socketface"
)

type initConfig struct {
	Mempool    pktmbuf.TemplateUpdates
	LCoreAlloc ealthread.AllocConfig
}

type parsedCommand struct {
	initCfg         initConfig
	tasks           []ping.TaskConfig
	counterInterval time.Duration
}

func parseCommand(args []string) (pc parsedCommand, e error) {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Var(yamlflag.New(&pc.initCfg), "initcfg", "initialization config object")
	flags.Var(yamlflag.New(&pc.tasks), "tasks", "ping task description")
	flags.DurationVar(&pc.counterInterval, "cnt", time.Second*10,
		"interval between printing counters (zero to disable)")

	e = flags.Parse(args)
	return pc, e
}
