package main

import (
	"fmt"
	"log"
	"os"

	"ndn-dpdk/dpdk"
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
var portRxQueues = make(map[dpdk.EthDev]dpdk.EthRxQueue)

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
		portRxQueues[port] = initEthDev(port)
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
	mp, e := dpdk.NewPktmbufPool(mpName, MP_CAPACITY, MP_CACHE, 0, MP_DATAROOM, socket)
	if e != nil {
		log.Printf("NewPktmbufPool(%d): %v", socket, e)
		os.Exit(EXIT_DPDK_ERROR)
	}
	mempools[socket] = mp
	return mp
}

func initEthDev(port dpdk.EthDev) dpdk.EthRxQueue {
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

	return rxQueues[0]
}

func slaveProc(port dpdk.EthDev) int {
	log.Printf("Processing %s on slave %d", port.GetName(), dpdk.GetCurrentLCore())
	rxq := portRxQueues[port]
	logger := log.New(os.Stdout, port.GetName()+" ", log.LstdFlags)

	pkts := make([]dpdk.Packet, RX_BURST_SIZE)
	for {
		burstSize := rxq.RxBurst(pkts)
		for _, pkt := range pkts[:burstSize] {
			processPacket(logger, pkt)
		}
	}

	return 0
}

func processPacket(logger *log.Logger, pkt dpdk.Packet) {
	defer pkt.Close()

	const ETHER_HDR_LEN = 14
	const MIN_NDN_PKT_LEN = 4
	if pkt.Len() < ETHER_HDR_LEN+MIN_NDN_PKT_LEN {
		return
	}
	pkt.GetFirstSegment().Adj(ETHER_HDR_LEN)

	d := ndn.NewTlvDecoder(pkt)
	switch {
	case d.IsInterest():
		interest, e := d.ReadInterest()
		if e != nil {
			return
		}
		logger.Printf("I %s", interest.GetName())
	case d.IsData():
		data, e := d.ReadData()
		if e != nil {
			return
		}
		logger.Printf("D %s", data.GetName())
	}
}
