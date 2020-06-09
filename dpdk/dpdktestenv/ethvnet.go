package dpdktestenv

import (
	"math/rand"

	"ndn-dpdk/dpdk"
)

// Configuration for EthVNet.
type EthVNetConfig struct {
	EthDevPairConfig
	NNodes int
}

func (cfg *EthVNetConfig) ApplyDefaults() {
	cfg.EthDevPairConfig.ApplyDefaults()
}

// A virtual Ethernet subnet.
type EthVNet struct {
	cfg EthVNetConfig

	Ports []dpdk.EthDev
	pairs []*EthDevPair

	NDrops      int
	bridgeLcore dpdk.LCore
	stop        chan bool
}

func NewEthVNet(cfg EthVNetConfig) (evn *EthVNet) {
	evn = new(EthVNet)
	evn.cfg = cfg
	evn.cfg.ApplyDefaults()
	evn.stop = make(chan bool)
	for i := 0; i < evn.cfg.NNodes; i++ {
		pair := NewEthDevPair(evn.cfg.EthDevPairConfig)
		pair.StartPortB()
		evn.pairs = append(evn.pairs, pair)
		evn.Ports = append(evn.Ports, pair.PortA)
	}
	return evn
}

func (evn *EthVNet) bridge() int {
	const BURST_SIZE = 25
	rxPkts := make([]dpdk.Packet, BURST_SIZE)
	for {
		for srcIndex, src := range evn.pairs {
			for _, srcQ := range src.RxqB {
				nRx := srcQ.RxBurst(rxPkts)

				for dstIndex, dst := range evn.pairs {
					if srcIndex == dstIndex {
						continue
					}

					var txPkts []dpdk.Packet
					for _, pkt := range rxPkts[:nRx] {
						txPkts = append(txPkts, packetFromBytesInMp(evn.cfg.MempoolId, pkt.ReadAll()))
					}

					dstQ := dst.TxqB[rand.Intn(len(dst.TxqB))]
					nTx := dstQ.TxBurst(txPkts)
					for i := nTx; i < len(txPkts); i++ {
						evn.NDrops++
						txPkts[i].Close()
					}
				}

				for _, rxPkt := range rxPkts[:nRx] {
					rxPkt.Close()
				}
			}
		}

		select {
		case <-evn.stop:
			return 0
		default:
		}
	}
}

func (evn *EthVNet) LaunchBridge(lcore dpdk.LCore) {
	evn.bridgeLcore = lcore
	lcore.RemoteLaunch(evn.bridge)
}

func (evn *EthVNet) Close() error {
	if evn.bridgeLcore.IsBusy() {
		evn.stop <- true
		evn.bridgeLcore.Wait()
	}
	for _, pair := range evn.pairs {
		pair.Close()
	}
	return nil
}
