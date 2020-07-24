package ethdev

import (
	"math/rand"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

// VNetConfig contains VNet configuration.
type VNetConfig struct {
	PairConfig
	NNodes int

	BurstSize       int
	LossProbability float64
	Shuffle         bool
}

func (cfg *VNetConfig) applyDefaults() {
	cfg.PairConfig.applyDefaults()
	if cfg.NNodes < 1 {
		cfg.NNodes = 1
	}
	if cfg.BurstSize < 1 {
		cfg.BurstSize = 25
	}
}

// VNet represents a virtual Ethernet subnet.
type VNet struct {
	ealthread.Thread // bridge thread

	cfg VNetConfig

	Ports []EthDev
	pairs []*Pair

	NDrops int
	stop   ealthread.StopChan
}

// NewVNet creates a virtual Ethernet subnet.
func NewVNet(cfg VNetConfig) *VNet {
	cfg.applyDefaults()
	vnet := &VNet{
		cfg:  cfg,
		stop: ealthread.NewStopChan(),
	}
	vnet.Thread = ealthread.New(
		cptr.Func0.Void(vnet.bridge),
		vnet.stop,
	)

	for i := 0; i < vnet.cfg.NNodes; i++ {
		pair := NewPair(vnet.cfg.PairConfig)
		pair.PortB.Start(pair.EthDevConfig())
		vnet.pairs = append(vnet.pairs, pair)
		vnet.Ports = append(vnet.Ports, pair.PortA)
	}
	return vnet
}

func (vnet *VNet) bridge() {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for vnet.stop.Continue() {
		for srcIndex, src := range vnet.pairs {
			for _, srcQ := range src.PortB.RxQueues() {
				rxPkts := make(pktmbuf.Vector, vnet.cfg.BurstSize)
				nRx := srcQ.RxBurst(rxPkts)
				if nRx == 0 {
					continue
				}

				if vnet.cfg.Shuffle {
					rng.Shuffle(nRx, func(i, j int) { rxPkts[i], rxPkts[j] = rxPkts[j], rxPkts[i] })
				}
				if rng.Float64() < vnet.cfg.LossProbability*float64(nRx) {
					nRx--
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
						txPkts[i].Append(pkt.Bytes())
					}

					dstQs := dst.PortB.TxQueues()
					dstQ := dstQs[rand.Intn(len(dstQs))]
					nTx := dstQ.TxBurst(txPkts)
					txDrops := txPkts[nTx:]
					vnet.NDrops += len(txDrops)
					txDrops.Close()
				}

				rxPkts.Close()
			}
		}
	}
}

// ThreadRole returns "VNETBRIDGE" used in lcore allocator.
func (*VNet) ThreadRole() string {
	return "VNETBRIDGE"
}

// Close stops the bridge and closes all ports.
func (vnet *VNet) Close() error {
	vnet.Stop()
	for _, pair := range vnet.pairs {
		pair.Close()
	}
	return nil
}
