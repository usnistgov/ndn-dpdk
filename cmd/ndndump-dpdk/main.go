package main

import (
	"fmt"
	"log"
	"os"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/ndn"
)

// exit codes
const (
	EXIT_ARG_ERROR  = 2
	EXIT_DPDK_ERROR = 3
)

// static configuration
const (
	MP_CAPACITY   = 255
	MP_CACHE      = 0
	MP_DATAROOM   = 2000
	RXQ_CAPACITY  = 64
	RX_BURST_SIZE = 8
)

var eal *dpdk.Eal
var mempools = make(map[dpdk.NumaSocket]dpdk.PktmbufPool)
var rxFaces = make(map[dpdk.EthDev]ethface.RxFace)

func main() {
	eal, e := dpdk.NewEal(os.Args)
	if e != nil {
		log.Print("NewEal:", e)
		os.Exit(EXIT_DPDK_ERROR)
	}

	ports := dpdk.ListEthDevs()
	if len(ports) > len(eal.Slaves) {
		log.Print("Number of slave lcores must be no less than number of NICs")
		os.Exit(EXIT_ARG_ERROR)
	}

	for _, port := range ports {
		rxFaces[port] = initEthDev(port)
	}

	for i, port := range ports {
		// TODO: port and slave should be on same NumaSocket
		slave := eal.Slaves[i]
		if !slave.RemoteLaunch(func() int { return slaveProc(port) }) {
			log.Printf("Failed to launch slave %d to process %s", slave, port.GetName())
			os.Exit(EXIT_DPDK_ERROR)
		}
	}

	for _, slave := range eal.Slaves {
		slave.Wait()
	}
}

func makeMempool(socket dpdk.NumaSocket) dpdk.PktmbufPool {
	if mp, ok := mempools[socket]; ok {
		return mp
	}

	mpName := fmt.Sprintf("MP_%d", socket)
	mp, e := dpdk.NewPktmbufPool(mpName, MP_CAPACITY, MP_CACHE,
		ndn.SizeofPacketPriv(), MP_DATAROOM, socket)
	if e != nil {
		log.Printf("NewPktmbufPool(%d): %v", socket, e)
		os.Exit(EXIT_DPDK_ERROR)
	}
	mempools[socket] = mp
	return mp
}

func initEthDev(port dpdk.EthDev) ethface.RxFace {
	socket := port.GetNumaSocket()
	mp := makeMempool(socket)

	var cfg dpdk.EthDevConfig
	cfg.AddRxQueue(dpdk.EthRxQueueConfig{Capacity: RXQ_CAPACITY, Socket: socket, Mp: mp})
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

	return ethface.NewRxFace(rxQueues[0])
}

func slaveProc(port dpdk.EthDev) int {
	log.Printf("Processing %s on slave %d", port.GetName(), dpdk.GetCurrentLCore())
	face := rxFaces[port]
	logger := log.New(os.Stdout, port.GetName()+" ", log.LstdFlags)

	pkts := make([]ndn.Packet, RX_BURST_SIZE)
	for {
		burstSize := face.RxBurst(pkts)
		for _, pkt := range pkts[:burstSize] {
			if !pkt.IsValid() {
				continue
			}
			processPacket(logger, pkt)
			pkt.Close()
		}
	}

	return 0
}

func processPacket(logger *log.Logger, pkt ndn.Packet) {
	switch pkt.GetNetType() {
	case ndn.NdnPktType_Interest:
		interest := pkt.AsInterest()
		logger.Printf("I %s", interest.GetName())
	case ndn.NdnPktType_Data:
		data := pkt.AsData()
		logger.Printf("D %s", data.GetName())
	}
}
