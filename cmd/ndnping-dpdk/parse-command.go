package main

import (
	"flag"
	"os"
	"time"

	"ndn-dpdk/app/ndnping"
	"ndn-dpdk/appinit"
)

type parsedCommand struct {
	initCfg             appinit.InitConfig
	tasks               []ndnping.TaskConfig
	counterInterval     time.Duration
	throughputBenchmark ThroughputBenchmarkConfig
}

func (pc parsedCommand) wantThroughputBenchmark() bool {
	return pc.throughputBenchmark.IntervalStep.Nanoseconds() > 0
}

func parseCommand(args []string) (pc parsedCommand, e error) {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	appinit.DeclareInitConfigFlag(flags, &pc.initCfg)
	appinit.DeclareConfigFlag(flags, &pc.tasks, "tasks", "ndnping task description")
	flags.DurationVar(&pc.counterInterval, "cnt", time.Second*10,
		"interval between printing counters (zero to disable)")
	appinit.DeclareConfigFlag(flags, &pc.throughputBenchmark, "throughput-benchmark", "throughput benchmark settings")

	e = flags.Parse(args)
	return pc, e
}
