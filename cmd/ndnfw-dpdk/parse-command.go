package main

import (
	"flag"
	"os"

	"ndn-dpdk/appinit"
)

type parsedCommand struct {
	initConfig appinit.InitConfig
}

func parseCommand(args []string) (pc parsedCommand, e error) {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	appinit.DeclareInitConfigFlag(flags, &pc.initConfig)

	e = flags.Parse(args)
	if e != nil {
		return pc, e
	}

	return pc, nil
}
