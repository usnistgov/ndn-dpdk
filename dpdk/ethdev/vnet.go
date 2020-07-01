package ethdev

import (
	"math/rand"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

// VNetConfig contains configuration for VNet.
type VNetConfig struct {
	PairConfig
	NNodes int
}

func (cfg *VNetConfig) applyDefaults() {
	cfg.PairConfig.applyDefaults()
	if cfg.NNodes < 1 {
		cfg.NNodes = 1
	}
}

// VNet represents a virtual Ethernet subnet.
type VNet struct {
	cfg VNetConfig

	Ports []EthDev
	pairs []*Pair

	NDrops      int
	bridgeLcore eal.LCore
	stop        chan bool
}

// NewVNet creates a virtual Ethernet subnet.
func NewVNet(cfg VNetConfig) (vnet *VNet) {
	vnet = new(VNet)
	cfg.applyDefaults()
	vnet.cfg = cfg
	vnet.stop = make(chan bool)
	for i := 0; i < vnet.cfg.NNodes; i++ {
		pair := NewPair(vnet.cfg.PairConfig)
		pair.PortB.Start(pair.EthDevConfig())
		vnet.pairs = append(vnet.pairs, pair)
		vnet.Ports = append(vnet.Ports, pair.PortA)
	}
	return vnet
}

func (vnet *VNet) bridge() int {
	const burstSize = 25
	for {
		for srcIndex, src := range vnet.pairs {
			for _, srcQ := range src.PortB.ListRxQueues() {
				rxPkts := make(pktmbuf.Vector, burstSize)
				nRx := srcQ.RxBurst(rxPkts)
				if nRx == 0 {
					continue
				}

				for dstIndex, dst := range vnet.pairs {
					if srcIndex == dstIndex {
						continue
					}

					txPkts, e := vnet.cfg.RxPool.Alloc(nRx)
					if e != nil {
						vnet.NDrops += nRx
						continue
					}
					for i, pkt := range rxPkts[:nRx] {
						txPkts[i].Append(pkt.ReadAll())
					}

					dstQs := dst.PortB.ListTxQueues()
					dstQ := dstQs[rand.Intn(len(dstQs))]
					nTx := dstQ.TxBurst(txPkts)
					txDrops := txPkts[nTx:]
					vnet.NDrops += len(txDrops)
					txDrops.Close()
				}

				rxPkts.Close()
			}
		}

		select {
		case <-vnet.stop:
			return 0
		default:
		}
	}
}

// LaunchBridge starts a bridge thread that copies packets between attached nodes.
func (vnet *VNet) LaunchBridge(lcore eal.LCore) {
	vnet.bridgeLcore = lcore
	lcore.RemoteLaunch(vnet.bridge)
}

// Close stops the bridge and closes all ports.
func (vnet *VNet) Close() error {
	if vnet.bridgeLcore.IsBusy() {
		vnet.stop <- true
		vnet.bridgeLcore.Wait()
	}
	for _, pair := range vnet.pairs {
		pair.Close()
	}
	return nil
}
