package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndnface"
)

// exit codes
const (
	EXIT_ARG_ERROR  = 2
	EXIT_DPDK_ERROR = 3
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

var eal *dpdk.Eal
var mp dpdk.PktmbufPool
var rxFace ndnface.RxFace
var txFaces []ndnface.TxFace

func main() {
	var e error
	eal, e = dpdk.NewEal(os.Args)
	if e != nil {
		log.Print("NewEal:", e)
		os.Exit(EXIT_DPDK_ERROR)
	}

	inIfname, outIfnames, e := parseCommand()
	ports := dpdk.ListEthDevs()
	inPort := dpdk.ETHDEV_INVALID
	var outPorts []dpdk.EthDev
	for _, port := range ports {
		ifname := port.GetName()
		switch {
		case ifname == inIfname:
			inPort = port
		case outIfnames[ifname]:
			outPorts = append(outPorts, port)
		}
	}
	if !inPort.IsValid() || len(outPorts) != len(outIfnames) {
		log.Print("Port not found")
		os.Exit(EXIT_ARG_ERROR)
	}
	log.Printf("inPort=%d outPorts=%v", inPort, outPorts)

	mp, e = dpdk.NewPktmbufPool("MP", MP_CAPACITY, MP_CACHE,
		ndn.SizeofPacketPriv(), MP_DATAROOM, inPort.GetNumaSocket())
	if e != nil {
		log.Printf("NewPktmbufPool: %v", e)
		os.Exit(EXIT_DPDK_ERROR)
	}

	rxFace = initRxFace(inPort)
	for _, port := range outPorts {
		txFaces = append(txFaces, initTxFace(port))
	}

	var inPkts [BURST_SIZE]ndn.Packet
	var outPkts [BURST_SIZE]ndn.Packet
	for {
		nRx := rxFace.RxBurst(inPkts[:])
		nTx := 0
		for _, pkt := range inPkts[:nRx] {
			if !pkt.IsValid() {
				continue
			}
			if pkt.Len() > 1488 {
				log.Print("packet over MTU, dropping")
				continue
			}
			outPkts[nTx] = pkt
			nTx++
		}
		if nRx > 0 {
			log.Printf("received %d, sending %d", nRx, nTx)
			for _, face := range txFaces {
				// TODO clone the packet before sending, because tx queue takes ownership
				nSent := face.TxBurst(outPkts[:nTx])
				log.Printf("sent %d", nSent)
			}
		}
	}
}

func parseCommand() (inface string, outfaces map[string]bool, e error) {
	var outfaceStr string

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.StringVar(&inface, "in", "", "input interface")
	flags.StringVar(&outfaceStr, "out", "", "output interface(s)")
	e = flags.Parse(eal.Args[1:])
	if e != nil {
		return
	}

	outfaces = make(map[string]bool)
	for _, f := range strings.Split(outfaceStr, ",") {
		outfaces[f] = true
	}
	return
}

func initRxFace(port dpdk.EthDev) ndnface.RxFace {
	var cfg dpdk.EthDevConfig
	cfg.AddRxQueue(dpdk.EthRxQueueConfig{Capacity: RXQ_CAPACITY,
		Socket: port.GetNumaSocket(), Mp: mp})
	rxQueues, _, e := port.Configure(cfg)
	if e != nil {
		log.Printf("port(%d).Configure: %v", port, e)
		os.Exit(EXIT_DPDK_ERROR)
	}

	port.SetPromiscuous(true)

	e = port.Start()
	if e != nil {
		log.Printf("port(%d).Start: %v", port, e)
		os.Exit(EXIT_DPDK_ERROR)
	}

	return ndnface.NewRxFace(rxQueues[0])
}

func initTxFace(port dpdk.EthDev) ndnface.TxFace {
	var cfg dpdk.EthDevConfig
	cfg.AddTxQueue(dpdk.EthTxQueueConfig{Capacity: TXQ_CAPACITY, Socket: port.GetNumaSocket()})
	_, txQueues, e := port.Configure(cfg)
	if e != nil {
		log.Printf("port(%d).Configure: %v", port, e)
		os.Exit(EXIT_DPDK_ERROR)
	}

	e = port.Start()
	if e != nil {
		log.Printf("port(%d).Start: %v", port, e)
		os.Exit(EXIT_DPDK_ERROR)
	}

	face, e := ndnface.NewTxFace(txQueues[0])
	if e != nil {
		log.Printf("NewTxFace(%d): %v", port, e)
		os.Exit(EXIT_DPDK_ERROR)
	}
	return face
}
