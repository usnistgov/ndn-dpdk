package main

import (
	"flag"
	"os"
	"time"

	"ndn-dpdk/app/ping"
	"ndn-dpdk/appinit"
)

type initConfig struct {
	appinit.InitConfig `yaml:",inline"`
	Ping               ping.InitConfig
}

type parsedCommand struct {
	initCfg         initConfig
	tasks           []ping.TaskConfig
	counterInterval time.Duration
}

func parseCommand(args []string) (pc parsedCommand, e error) {
	pc.initCfg.Ping.QueueCapacity = 256

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	appinit.DeclareInitConfigFlag(flags, &pc.initCfg)
	appinit.DeclareConfigFlag(flags, &pc.tasks, "tasks", "ping task description")
	flags.DurationVar(&pc.counterInterval, "cnt", time.Second*10,
		"interval between printing counters (zero to disable)")

	e = flags.Parse(args)
	return pc, e
}
