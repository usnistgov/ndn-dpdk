package main

import (
	"flag"
	"os"
	"strings"
	"time"

	"ndn-dpdk/appinit"
)

type parsedCommand struct {
	inface          string
	outfaces        []string
	wantDump        bool
	counterInterval time.Duration
}

func parseCommand() (pc parsedCommand, e error) {
	var outfaceStr string

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.StringVar(&pc.inface, "in", "", "input face")
	flags.StringVar(&outfaceStr, "out", "", "output face(s)")
	flags.BoolVar(&pc.wantDump, "dump", false, "log every packet")
	flags.DurationVar(&pc.counterInterval, "cnt", time.Second*10, "interval between printing counters")

	e = flags.Parse(appinit.Eal.Args[1:])
	if e != nil {
		return
	}

	pc.outfaces = strings.Split(outfaceStr, ",")
	if len(pc.outfaces) == 1 && pc.outfaces[0] == "" {
		pc.outfaces = pc.outfaces[:0]
	}
	return
}
