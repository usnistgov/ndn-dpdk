package main

import (
	"flag"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/ping"
	"github.com/usnistgov/ndn-dpdk/core/yamlflag"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"

	_ "github.com/usnistgov/ndn-dpdk/iface/ethface"
	_ "github.com/usnistgov/ndn-dpdk/iface/socketface"
)

type initConfig struct {
	Eal        ealconfig.Config `json:"eal"`
	Mempool    pktmbuf.TemplateUpdates
	LCoreAlloc ealthread.AllocConfig
}

func parseCommand() (cfg initConfig, tasks []ping.TaskConfig, counterInterval time.Duration) {
	flag.Var(yamlflag.New(&cfg), "initcfg", "initialization config object")
	flag.Var(yamlflag.New(&tasks), "tasks", "ping task description")
	flag.DurationVar(&counterInterval, "cnt", time.Second*10,
		"interval between printing counters (zero to disable)")

	flag.Parse()
	return
}
