package dpdktestenv

import "ndn-dpdk/dpdk"

// A virtual Ethernet link.
// EthDevPair directly transfers TX mbufs to the RX queue, which can cause undesired
// side-effects if private data is attached to those mbufs. EthVLink copies the payload
// into new mbufs. It is slower but behaves more like a real link.
type EthVLink struct {
	PortA dpdk.EthDev
	RxqA  dpdk.EthRxQueue // first RX queue of port A
	TxqA  dpdk.EthTxQueue // first TX queue of port A

	PortB dpdk.EthDev
	RxqB  dpdk.EthRxQueue // first RX queue of port B
	TxqB  dpdk.EthTxQueue // first TX queue of port B

	AtoB []int // per-queue packet count from A to B
	BtoA []int // per-queue packet count from B to A

	mpid  string
	pairA *EthDevPair
	pairB *EthDevPair
	stop  chan struct{}
}

func NewEthVLink(nQueues, ringCapacity, queueCapacity int, mpid string) *EthVLink {
	var evl EthVLink

	evl.mpid = mpid
	evl.pairA = NewEthDevPair(nQueues, ringCapacity, queueCapacity)
	evl.pairB = NewEthDevPair(nQueues, ringCapacity, queueCapacity)
	evl.stop = make(chan struct{}, 1)

	evl.PortA = evl.pairA.PortA
	evl.RxqA = evl.pairA.RxqA[0]
	evl.TxqA = evl.pairA.TxqA[0]
	evl.PortB = evl.pairB.PortB
	evl.RxqB = evl.pairB.RxqB[0]
	evl.TxqB = evl.pairB.TxqB[0]

	evl.AtoB = make([]int, nQueues)
	evl.BtoA = make([]int, nQueues)

	return &evl
}

func (evl *EthVLink) Bridge() int {
	const BURST_SIZE = 64
	rxPkts := make([]dpdk.Packet, BURST_SIZE)
	txPkts := make([]dpdk.Packet, BURST_SIZE)

	transfer := func(rxq dpdk.EthRxQueue, txq dpdk.EthTxQueue) int {
		nRx := rxq.RxBurst(rxPkts)
		for i, rxPkt := range rxPkts[:nRx] {
			txPkts[i] = packetFromBytesInMp(evl.mpid, rxPkt.ReadAll())
			rxPkt.Close()
		}
		txq.TxBurst(txPkts[:nRx])
		return nRx
	}

	for {
		for i := range evl.AtoB {
			evl.AtoB[i] += transfer(evl.pairA.RxqB[i], evl.pairB.TxqA[i])
		}
		for i := range evl.BtoA {
			evl.BtoA[i] += transfer(evl.pairB.RxqA[i], evl.pairA.TxqB[i])
		}

		select {
		case <-evl.stop:
			return 0
		default:
		}
	}
}

func (evl *EthVLink) Close() error {
	evl.stop <- struct{}{}
	evl.pairA.Close()
	evl.pairB.Close()
	return nil
}
