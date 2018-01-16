package main

import (
	"log"
	"time"

	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

const DISCARD_BURST_SIZE = 8

func main() {
	appinit.InitEal()
	pc, e := parseCommand(appinit.Eal.Args[1:])
	if e != nil {
		appinit.Exitf(appinit.EXIT_BAD_CONFIG, "parseCommand: %v", e)
	}

	rxFace, e := appinit.NewFaceFromUri(pc.inface)
	if e != nil {
		appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewFaceFromUri(%s): %v", pc.inface, e)
	}
	pcrx, e := NewPktcopyRx(*rxFace)
	if e != nil {
		appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewPktcopyRx(%d): %v", rxFace.GetFaceId(), e)
	}

	var txFaces []iface.Face
	var pctxs []PktcopyTx
	txFacesByNumaSocket := make(map[dpdk.NumaSocket][]iface.Face)
	for _, outface := range pc.outfaces {
		txFace, e := appinit.NewFaceFromUri(outface)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewFaceFromUri(%s): %v", outface, e)
		}
		txFaces = append(txFaces, *txFace)
		numaSocket := txFace.GetNumaSocket()
		txFacesByNumaSocket[numaSocket] = append(txFacesByNumaSocket[numaSocket], *txFace)

		pctx, e := NewPktcopyTx(*txFace)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewPktcopyTx(%d): %v", txFace.GetFaceId(), e)
		}
		pcrx.LinkTo(pctx)
		pctxs = append(pctxs, pctx)
	}

	tick := time.Tick(pc.counterInterval)
	go func() {
		for {
			<-tick
			log.Printf("RX-cnt %d %v", rxFace.GetFaceId(), rxFace.ReadCounters())
			for _, txFace := range txFaces {
				log.Printf("TX-cnt %d %v", txFace.GetFaceId(), txFace.ReadCounters())
			}
		}
	}()

	// start PktcopyTx processes
	for _, pctx := range pctxs {
		appinit.LaunchRequired(pctx.Run, pctx.GetFace().GetNumaSocket())
	}

	// start PktcopyRx process
	appinit.LaunchRequired(pcrx.Run, pcrx.GetFace().GetNumaSocket())

	for numaSocket := range txFacesByNumaSocket {
		txFacesOnSocket := txFacesByNumaSocket[numaSocket]
		// discard packets arrived on outfaces
		appinit.LaunchRequired(func() int {
			for {
				for _, txFace := range txFacesOnSocket {
					var pkts [DISCARD_BURST_SIZE]ndn.Packet
					nPkts := txFace.RxBurst(pkts[:])
					if nPkts == 0 {
						continue
					}
					for _, pkt := range pkts[:nPkts] {
						pkt.Close()
					}
				}
			}
		}, numaSocket)
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
