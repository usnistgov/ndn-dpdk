package main

import (
	"log"
	"time"

	"ndn-dpdk/appinit"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

func main() {
	appinit.InitEal()
	pc, e := ParseCommand(appinit.Eal.Args[1:])
	if e != nil {
		appinit.Exitf(appinit.EXIT_BAD_CONFIG, "parseCommand: %v", e)
	}

	var faces []iface.Face
	var pcrxs []PktcopyRx
	var pctxs []PktcopyTx

	for _, faceUri := range pc.Faces {
		face, e := appinit.NewFaceFromUri(faceUri)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewFaceFromUri(%s): %v", faceUri, e)
		}
		faces = append(faces, *face)

		pcrx, e := NewPktcopyRx(*face)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewPktcopyRx(%d): %v", face.GetFaceId(), e)
		}
		pcrxs = append(pcrxs, pcrx)

		pctx, e := NewPktcopyTx(*face)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewPktcopyTx(%d): %v", face.GetFaceId(), e)
		}
		pctxs = append(pctxs, pctx)
	}

	// link PktcopyRx and PktcopyTx
	switch pc.Mode {
	case TopoMode_Pair:
		for i := 0; i < len(faces); i += 2 {
			pcrxs[i].LinkTo(pctxs[i+1])
			pcrxs[i+1].LinkTo(pctxs[i])
		}
	case TopoMode_All:
		for i, pcrx := range pcrxs {
			for j, pctx := range pctxs {
				if i == j {
					continue
				}
				pcrx.LinkTo(pctx)
			}
		}
	case TopoMode_OneWay:
		for _, pctx := range pctxs[1:] {
			pcrxs[0].LinkTo(pctx)
		}
	}

	// print counters
	tick := time.Tick(pc.CntInterval)
	go func() {
		for {
			<-tick
			for _, face := range faces {
				log.Printf("%d %v", face.GetFaceId(), face.ReadCounters())
			}
		}
	}()

	// start PktcopyTx processes
	for _, pctx := range pctxs {
		appinit.LaunchRequired(pctx.Run, pctx.GetFace().GetNumaSocket())
	}

	// start PktcopyRx process
	for _, pcrx := range pcrxs {
		appinit.LaunchRequired(pcrx.Run, pcrx.GetFace().GetNumaSocket())
	}

	select {}
}

func printPacket(pkt ndn.Packet) {
	switch pkt.GetNetType() {
	case ndn.NdnPktType_Interest:
		interest := pkt.AsInterest()
		log.Printf("I %s", interest.GetName())
	case ndn.NdnPktType_Data:
		data := pkt.AsData()
		log.Printf("D %s", data.GetName())
	case ndn.NdnPktType_Nack:
		log.Printf("Nack")
	}
}
