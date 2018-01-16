package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"ndn-dpdk/iface/faceuri"
)

type TopoMode int

const (
	TopoMode_Pair TopoMode = iota
	TopoMode_All
	TopoMode_OneWay
)

type ParsedCommand struct {
	Mode        TopoMode
	Faces       []faceuri.FaceUri
	Dump        bool
	CntInterval time.Duration
}

func ParseCommand(args []string) (pc ParsedCommand, e error) {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	faceUris := flags.String("faces", "", "face(s)")
	flags.Bool("pair", true, "forward between face at index 0-1, 2-3, 4-5, etc")
	allMode := flags.Bool("all", false, "copy packets among all faces")
	onewayMode := flags.Bool("oneway", false, "receive on first face and send on all other faces")
	flags.BoolVar(&pc.Dump, "dump", false, "log every packet")
	flags.DurationVar(&pc.CntInterval, "cnt", time.Second*10, "interval between printing counters")

	e = flags.Parse(args)
	if e != nil {
		return pc, e
	}

	switch {
	case *allMode:
		pc.Mode = TopoMode_All
	case *onewayMode:
		pc.Mode = TopoMode_OneWay
	default:
		pc.Mode = TopoMode_Pair
	}

	foundFaceUris := make(map[string]bool)
	for _, faceUri := range strings.Split(*faceUris, ",") {
		if faceUri == "" {
			continue
		}
		u, e := faceuri.Parse(faceUri)
		if e != nil {
			return pc, e
		}
		normailzedUri := u.String()
		if foundFaceUris[normailzedUri] {
			return pc, fmt.Errorf("duplicate FaceUri %s (normalized as %v)", faceUri, normailzedUri)
		}
		foundFaceUris[normailzedUri] = true
		pc.Faces = append(pc.Faces, *u)
	}

	return pc, nil
}
