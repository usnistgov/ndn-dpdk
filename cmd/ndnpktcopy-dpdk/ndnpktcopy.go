package main

import (
	"log"
	"os"
	"time"

	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

// static configuration
const (
	MP_CAPACITY  = 255
	MP_CACHE     = 0
	MP_DATAROOM  = 2000
	RXQ_CAPACITY = 64
	TXQ_CAPACITY = 64
	BURST_SIZE   = 8
)

var rxFace *iface.Face
var txFaces []*iface.Face
var txFacesByNumaSocket = make(map[dpdk.NumaSocket][]*iface.Face)

func main() {
	pc, e := parseCommand()

	rxFace, _, e = createFaceFromUri(pc.inface)
	if e != nil {
		log.Printf("createFaceFromUri(%s): %v", pc.inface, e)
		os.Exit(appinit.EXIT_FACE_INIT_ERROR)
	}

	for _, outface := range pc.outfaces {
		txFace, isNew, e := createFaceFromUri(outface)
		if e != nil {
			log.Printf("createFaceFromUri(%s): %v", outface, e)
			os.Exit(appinit.EXIT_FACE_INIT_ERROR)
		}
		if !isNew {
			log.Printf("duplicate face %s", outface)
			os.Exit(appinit.EXIT_BAD_CONFIG)
		}
		numaSocket := txFace.GetNumaSocket()
		txFaces = append(txFaces, txFace)
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
