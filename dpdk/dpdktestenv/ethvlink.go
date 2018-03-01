package dpdktestenv

import "ndn-dpdk/dpdk"

// A virtual Ethernet link.
// EthDevPair directly transfers TX mbufs to the RX queue, which can cause undesired
// side-effects if private data is attached to those mbufs. EthVLink copies the payload
// into new mbufs. It is slower but behaves more like a real link.
type EthVLink struct {
	PortA dpdk.EthDev
	RxqA  dpdk.EthRxQueue
	TxqA  dpdk.EthTxQueue
	PortB dpdk.EthDev
	RxqB  dpdk.EthRxQueue
	TxqB  dpdk.EthTxQueue

	mpid  string
	pairA *EthDevPair
	pairB *EthDevPair
	stop  chan struct{}
}

func NewEthVLink(ringCapacity int, queueCapacity int, mpid string) *EthVLink {
	var evl EthVLink

	evl.mpid = mpid
	evl.pairA = NewEthDevPair(1, ringCapacity, queueCapacity)
	evl.pairB = NewEthDevPair(1, ringCapacity, queueCapacity)
	evl.stop = make(chan struct{}, 1)

	evl.PortA = evl.pairA.PortA
	evl.RxqA = evl.pairA.RxqA[0]
	evl.TxqA = evl.pairA.TxqA[0]
	evl.PortB = evl.pairB.PortB
	evl.RxqB = evl.pairB.RxqB[0]
	evl.TxqB = evl.pairB.TxqB[0]

	return &evl
}

func (evl *EthVLink) Bridge() int {
	const BURST_SIZE = 64
	rxPkts := make([]dpdk.Packet, BURST_SIZE)
	txPkts := make([]dpdk.Packet, BURST_SIZE)

	transfer := func(rxq dpdk.EthRxQueue, txq dpdk.EthTxQueue) {
		nRx := rxq.RxBurst(rxPkts)
		for i, rxPkt := range rxPkts[:nRx] {
			txPkts[i] = packetFromBytesInMp(evl.mpid, rxPkt.ReadAll())
			rxPkt.Close()
		}
		txq.TxBurst(txPkts[:nRx])
	}

	for {
		transfer(evl.pairA.RxqB[0], evl.pairB.TxqA[0])
		transfer(evl.pairB.RxqA[0], evl.pairA.TxqB[0])

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
