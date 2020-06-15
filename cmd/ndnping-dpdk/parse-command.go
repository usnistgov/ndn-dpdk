package main

import (
	"flag"
	"os"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/ping"
	"github.com/usnistgov/ndn-dpdk/appinit"
	"github.com/usnistgov/ndn-dpdk/iface/createface"
)

type parsedCommand struct {
	initCfg         appinit.InitConfig
	tasks           []ping.TaskConfig
	counterInterval time.Duration
}

func parseCommand(args []string) (pc parsedCommand, e error) {
	pc.initCfg.Face = createface.GetDefaultConfig()

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	appinit.DeclareInitConfigFlag(flags, &pc.initCfg)
	appinit.DeclareConfigFlag(flags, &pc.tasks, "tasks", "ping task description")
	flags.DurationVar(&pc.counterInterval, "cnt", time.Second*10,
		"interval between printing counters (zero to disable)")

	e = flags.Parse(args)
	return pc, e
}
