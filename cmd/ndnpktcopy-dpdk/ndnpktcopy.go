package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"ndn-dpdk/app/dump"
	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

const Dump_RingCapacity = 64

func main() {
	appinit.InitEal()
	pc, e := ParseCommand(appinit.Eal.Args[1:])
	if e != nil {
		appinit.Exitf(appinit.EXIT_BAD_CONFIG, "parseCommand: %v", e)
	}

	var pcrxs []PktcopyRx
	var pctxs []PktcopyTx
	var dumps []dump.Dump

	for _, faceUri := range pc.Faces {
		face, e := appinit.NewFaceFromUri(faceUri)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewFaceFromUri(%s): %v", faceUri, e)
		}

		pcrx, e := NewPktcopyRx(face)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewPktcopyRx(%d): %v", face.GetFaceId(), e)
		}
		pcrxs = append(pcrxs, pcrx)

		pctx, e := NewPktcopyTx(face)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewPktcopyTx(%d): %v", face.GetFaceId(), e)
		}
		pctxs = append(pctxs, pctx)
	}

	// enable dump
	if pc.Dump {
		for i, pcrx := range pcrxs {
			ringName := fmt.Sprintf("dump_%d", i)
			ring, e := dpdk.NewRing(ringName, Dump_RingCapacity, pcrx.GetFace().GetNumaSocket(), true, true)
			if e != nil {
				appinit.Exitf(appinit.EXIT_RING_INIT_ERROR, "NewRing(%s): %v", ringName, e)
			}
			pcrx.LinkTo(ring)

			prefix := fmt.Sprintf("%d ", pcrx.GetFace().GetFaceId())
			logger := log.New(os.Stderr, prefix, log.Lmicroseconds)
			dumper := dump.New(ring, logger)
			dumps = append(dumps, dumper)
		}
	}

	// link PktcopyRx and PktcopyTx
	switch pc.Mode {
	case TopoMode_Pair:
		for i := 0; i < len(pcrxs); i += 2 {
			pcrxs[i].LinkTo(pctxs[i+1].GetRing())
			pcrxs[i+1].LinkTo(pctxs[i].GetRing())
		}
	case TopoMode_All:
		for i, pcrx := range pcrxs {
			for j, pctx := range pctxs {
				if i == j {
					continue
				}
				pcrx.LinkTo(pctx.GetRing())
			}
		}
	case TopoMode_OneWay:
		for _, pctx := range pctxs[1:] {
			pcrxs[0].LinkTo(pctx.GetRing())
		}
	}

	// print counters
	tick := time.Tick(pc.CntInterval)
	go func() {
		for {
			<-tick
			for _, faceId := range iface.ListFaceIds() {
				log.Printf("%d %v", faceId, iface.Get(faceId).ReadCounters())
			}
		}
	}()

	// start PktcopyTx processes
	for _, pctx := range pctxs {
		appinit.LaunchRequired(pctx.Run, pctx.GetFace().GetNumaSocket())
	}

	// start PktcopyRx processes
	for _, pcrx := range pcrxs {
		appinit.LaunchRequired(pcrx.Run, pcrx.GetFace().GetNumaSocket())
	}

	// start Dump processes
	for _, dump := range dumps {
		appinit.LaunchRequired(dump.Run, dpdk.NUMA_SOCKET_ANY)
	}

	select {}
}
