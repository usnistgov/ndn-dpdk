package main

import (
	"flag"
	"os"
	"time"

	"ndn-dpdk/app/ping"
	"ndn-dpdk/appinit"
	"ndn-dpdk/iface/createface"
)

type initConfig struct {
	appinit.InitConfig
	Ping ping.InitConfig
}

type parsedCommand struct {
	initCfg         initConfig
	tasks           []ping.TaskConfig
	counterInterval time.Duration
}

func parseCommand(args []string) (pc parsedCommand, e error) {
	pc.initCfg.Face = createface.GetDefaultConfig()
	pc.initCfg.Ping.QueueCapacity = 256

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	appinit.DeclareInitConfigFlag(flags, &pc.initCfg)
	appinit.DeclareConfigFlag(flags, &pc.tasks, "tasks", "ping task description")
	flags.DurationVar(&pc.counterInterval, "cnt", time.Second*10,
		"interval between printing counters (zero to disable)")

	e = flags.Parse(args)
	return pc, e
}
