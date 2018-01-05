package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"ndn-dpdk/iface/faceuri"
)

type parsedCommand struct {
	inface          faceuri.FaceUri
	outfaces        []faceuri.FaceUri
	wantDump        bool
	counterInterval time.Duration
}

func parseCommand(args []string) (pc parsedCommand, e error) {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	inface := flags.String("in", "", "input face")
	outfaces := flags.String("out", "", "output face(s)")
	flags.BoolVar(&pc.wantDump, "dump", false, "log every packet")
	flags.DurationVar(&pc.counterInterval, "cnt", time.Second*10, "interval between printing counters")

	e = flags.Parse(args)
	if e != nil {
		return pc, e
	}

	foundFaceUris := make(map[string]bool)

	u, e := faceuri.Parse(*inface)
	if e != nil {
		return pc, e
	}
	pc.inface = *u
	foundFaceUris[u.String()] = true

	for _, outface := range strings.Split(*outfaces, ",") {
		if outface == "" {
			continue
		}
		u, e = faceuri.Parse(outface)
		if e != nil {
			return pc, e
		}
		normailzedUri := u.String()
		if foundFaceUris[normailzedUri] {
			return pc, fmt.Errorf("duplicate FaceUri %s (normailzed as %v)", outface, normailzedUri)
		}
		foundFaceUris[normailzedUri] = true
		pc.outfaces = append(pc.outfaces, *u)
	}
	return pc, nil
}
