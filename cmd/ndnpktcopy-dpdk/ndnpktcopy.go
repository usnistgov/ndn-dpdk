package main

import (
	"log"
	"time"

	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

const BURST_SIZE = 8

var rxFace *iface.Face
var txFaces []*iface.Face
var txFacesByNumaSocket = make(map[dpdk.NumaSocket][]*iface.Face)

func main() {
	appinit.InitEal()
	pc, e := parseCommand(appinit.Eal.Args[1:])
	if e != nil {
		appinit.Exitf(appinit.EXIT_BAD_CONFIG, "parseCommand: %v", e)
	}

	rxFace, e = appinit.NewFaceFromUri(pc.inface)
	if e != nil {
		appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "createFaceFromUri(%s): %v", pc.inface, e)
	}

	for _, outface := range pc.outfaces {
		txFace, e := appinit.NewFaceFromUri(outface)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "createFaceFromUri(%s): %v", outface, e)
		}
		txFaces = append(txFaces, txFace)
		numaSocket := txFace.GetNumaSocket()
		txFacesByNumaSocket[numaSocket] = append(txFacesByNumaSocket[numaSocket], txFace)
	}

	tick := time.Tick(pc.counterInterval)
	go func() {
		for {
			<-tick
			log.Printf("RX-cnt %d %v", rxFace.GetFaceId(), rxFace.ReadCounters())
			for _, txFace := range txFaces {
				log.Printf("TX-cnt %d %v", txFace.GetFaceId(), txFace.ReadCounters())
			}
			// log.Printf("MP-usage RX=%d TXHDR=%d IND=%d", mpRx.CountInUse(),
			// 	mpTxHdr.CountInUse(), mpIndirect.CountInUse())
		}
	}()

	// copy packets from inface to outface
	appinit.LaunchRequired(func() int {
		for {
			var pkts [BURST_SIZE]ndn.Packet
			nPkts := rxFace.RxBurst(pkts[:])
			if nPkts == 0 {
				continue
			}
			for _, txFace := range txFaces {
				txFace.TxBurst(pkts[:nPkts])
			}
			for _, pkt := range pkts[:nPkts] {
				if pc.wantDump {
					printPacket(pkt)
				}
				pkt.Close()
			}
		}
	}, rxFace.GetNumaSocket())

	for numaSocket := range txFacesByNumaSocket {
		txFacesOnSocket := txFacesByNumaSocket[numaSocket]
		// discard packets arrived on outfaces
		appinit.LaunchRequired(func() int {
			for {
				for _, txFace := range txFacesOnSocket {
					var pkts [BURST_SIZE]ndn.Packet
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
