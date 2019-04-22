package dpdktestenv

import (
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
	const BURST_SIZE = 64
	rxPkts := make([]dpdk.Packet, BURST_SIZE)
	txPkts := make([]dpdk.Packet, BURST_SIZE)

	for {
		for i, src := range evn.pairs {
			nRx := src.RxqB[0].RxBurst(rxPkts)
			for j, dst := range evn.pairs {
				if i == j {
					continue
				}
				for k, rxPkt := range rxPkts[:nRx] {
					txPkts[k] = packetFromBytesInMp(evn.cfg.MempoolId, rxPkt.ReadAll())
				}
				dst.TxqB[0].TxBurst(txPkts[:nRx])
			}
			for _, rxPkt := range rxPkts[:nRx] {
				rxPkt.Close()
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
	evn.stop <- true
	evn.bridgeLcore.Wait()
	for _, pair := range evn.pairs {
		pair.Close()
	}
	return nil
}
