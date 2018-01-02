package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strings"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/socketface"
	"ndn-dpdk/ndn"
)

// exit codes
const (
	EXIT_ARG_ERROR    = 2
	EXIT_DPDK_ERROR   = 3
	EXIT_SOCKET_ERROR = 4
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
var mpRx dpdk.PktmbufPool
var mpTxHdr dpdk.PktmbufPool
var mpIndirect dpdk.PktmbufPool
var rxFace iface.Face
var txFaces []iface.Face

func main() {
	var e error
	eal, e = dpdk.NewEal(os.Args)
	if e != nil {
		log.Print("NewEal:", e)
		os.Exit(EXIT_DPDK_ERROR)
	}

	pc, e := parseCommand()
	ports := dpdk.ListEthDevs()
	inPort := dpdk.ETHDEV_INVALID
	var outPorts []dpdk.EthDev
	for _, port := range ports {
		ifname := port.GetName()
		switch {
		case ifname == pc.inface:
			inPort = port
		case pc.outfaces[ifname]:
			outPorts = append(outPorts, port)
		}
	}
	if !inPort.IsValid() || len(outPorts) != len(pc.outfaces) {
		log.Print("Port not found")
		os.Exit(EXIT_ARG_ERROR)
	}
	log.Printf("inPort=%d outPorts=%v", inPort, outPorts)

	mpRx, e = dpdk.NewPktmbufPool("MP-RX", MP_CAPACITY, MP_CACHE,
		ndn.SizeofPacketPriv(), MP_DATAROOM, inPort.GetNumaSocket())
	if e != nil {
		log.Printf("NewPktmbufPool(RX): %v", e)
		os.Exit(EXIT_DPDK_ERROR)
	}

	mpTxHdr, e = dpdk.NewPktmbufPool("MP-TXHDR", MP_CAPACITY, MP_CACHE,
		0, ethface.SizeofHeaderMempoolDataRoom(), outPorts[0].GetNumaSocket())
	if e != nil {
		log.Printf("NewPktmbufPool(TXHDR): %v", e)
		os.Exit(EXIT_DPDK_ERROR)
	}

	mpIndirect, e = dpdk.NewPktmbufPool("MP-IND", MP_CAPACITY, MP_CACHE,
		0, 0, outPorts[0].GetNumaSocket())
	if e != nil {
		log.Printf("NewPktmbufPool(IND): %v", e)
		os.Exit(EXIT_DPDK_ERROR)
	}

	if len(pc.outsock) > 0 {
		outsockParams := strings.SplitN(pc.outsock, ":", 2)
		if len(outsockParams) != 2 {
			log.Print("-connect syntax error")
			os.Exit(EXIT_ARG_ERROR)
		}
		network, address := outsockParams[0], outsockParams[1]
		face := initSocketFace(network, address)
		txFaces = append(txFaces, face)
	}

	rxFace = initEthFace(inPort)
	for _, port := range outPorts {
		txFaces = append(txFaces, initEthFace(port))
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
			outPkts[nTx] = pkt
			nTx++
		}
		if nRx > 0 {
			for _, face := range txFaces {
				face.TxBurst(outPkts[:nTx])
			}
			for _, pkt := range outPkts[:nTx] {
				printPacket(pkt)
				pkt.Close()
			}
		}
	}
}

type parsedCommand struct {
	inface   string
	outfaces map[string]bool
	outsock  string
}

func parseCommand() (pc parsedCommand, e error) {
	var outfaceStr string

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.StringVar(&pc.inface, "in", "", "input interface")
	flags.StringVar(&outfaceStr, "out", "", "output interface(s)")
	flags.StringVar(&pc.outsock, "connect", "", "output socket")
	e = flags.Parse(eal.Args[1:])
	if e != nil {
		return
	}

	pc.outfaces = make(map[string]bool)
	for _, f := range strings.Split(outfaceStr, ",") {
		pc.outfaces[f] = true
	}
	return
}

func initEthFace(port dpdk.EthDev) iface.Face {
	var cfg dpdk.EthDevConfig
	cfg.AddRxQueue(dpdk.EthRxQueueConfig{Capacity: RXQ_CAPACITY,
		Socket: port.GetNumaSocket(), Mp: mpRx})
	cfg.AddTxQueue(dpdk.EthTxQueueConfig{Capacity: TXQ_CAPACITY, Socket: port.GetNumaSocket()})
	_, _, e := port.Configure(cfg)
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

	face, e := ethface.New(port, mpIndirect, mpTxHdr)
	if e != nil {
		log.Printf("ethface.New(%d): %v", port, e)
		os.Exit(EXIT_DPDK_ERROR)
	}
	return face.Face
}

func initSocketFace(network, address string) iface.Face {
	conn, e := net.Dial(network, address)
	if e != nil {
		log.Printf("net.Dial(%s,%s): %v", network, address, e)
		os.Exit(EXIT_SOCKET_ERROR)
	}

	var cfg socketface.Config
	cfg.RxMp = mpRx
	cfg.RxqCapacity = RXQ_CAPACITY
	cfg.TxqCapacity = TXQ_CAPACITY

	face := socketface.New(conn, cfg)
	return face.Face
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
