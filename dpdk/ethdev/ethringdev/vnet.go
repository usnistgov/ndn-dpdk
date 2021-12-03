package ethringdev

import (
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"go.uber.org/zap"
	"go4.org/must"
)

// VNetConfig contains VNet configuration.
type VNetConfig struct {
	PairConfig
	NNodes int // number of nodes

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

type vnetPair struct {
	*Pair
	rxq []ethdev.RxQueue // bridge-side RX queues
	txq []ethdev.TxQueue // bridge-side TX queues
}

// VNet represents a simulated Ethernet subnet.
type VNet struct {
	ealthread.Thread // bridge thread

	cfg    VNetConfig
	logger *zap.Logger
	rng    *rand.Rand
	stop   ealthread.StopChan

	pairs []vnetPair

	Ports  []ethdev.EthDev // app-side EthDev
	NDrops int             // number of dropped packets
}

func (vnet *VNet) bridge() {
	for vnet.stop.Continue() {
		for srcIndex, src := range vnet.pairs {
			for _, srcQ := range src.rxq {
				vnet.pass(srcIndex, srcQ)
			}
		}
	}
}

func (vnet *VNet) pass(srcIndex int, srcQ ethdev.RxQueue) {
	rxPkts := make(pktmbuf.Vector, vnet.cfg.BurstSize)
	nRx := srcQ.RxBurst(rxPkts)
	if nRx == 0 {
		return
	}
	defer must.Close(rxPkts)

	if vnet.cfg.Shuffle {
		vnet.rng.Shuffle(nRx, reflect.Swapper(rxPkts))
	}
	if vnet.rng.Float64() < vnet.cfg.LossProbability*float64(nRx) {
		nRx--
	}

	for dstIndex, dst := range vnet.pairs {
		if srcIndex == dstIndex {
			continue
		}

		txPkts, e := vnet.cfg.RxPool.Alloc(nRx)
		if e != nil {
			vnet.logger.Warn("vnet alloc error",
				zap.Int("count", nRx),
				zap.Int("rxpool-avail", vnet.cfg.RxPool.CountAvailable()),
			)
			vnet.NDrops += nRx
			continue
		}
		for i, pkt := range rxPkts[:nRx] {
			txPkts[i].Append(pkt.Bytes())
		}

		dstQ := dst.txq[vnet.rng.Intn(len(dst.txq))]
		nTx := dstQ.TxBurst(txPkts)
		txDrops := txPkts[nTx:]
		vnet.NDrops += len(txDrops)
		must.Close(txDrops)
	}
}

// ThreadRole returns "VNETBRIDGE" used in lcore allocator.
func (*VNet) ThreadRole() string {
	return "VNETBRIDGE"
}

// Close stops the bridge and closes all ports.
func (vnet *VNet) Close() error {
	e := vnet.Stop()
	for _, pair := range vnet.pairs {
		must.Close(pair)
	}
	return e
}

// NewVNet creates a virtual Ethernet subnet.
func NewVNet(cfg VNetConfig) (vnet *VNet, e error) {
	cfg.applyDefaults()
	vnet = &VNet{
		cfg:    cfg,
		logger: logger.With(zap.Int("vnet", rand.Int())),
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
		stop:   ealthread.NewStopChan(),
	}
	vnet.Thread = ealthread.New(
		cptr.Func0.Void(vnet.bridge),
		vnet.stop,
	)

	ports, bridgePorts := []int{}, []int{}
	for i := 0; i < cfg.NNodes; i++ {
		pair, e := NewPair(cfg.PairConfig)
		if e != nil {
			must.Close(vnet)
			return nil, fmt.Errorf("ethringdev.NewPair %w", e)
		}
		pair.PortB.Start(pair.EthDevConfig())
		vnet.pairs = append(vnet.pairs, vnetPair{
			Pair: pair,
			rxq:  pair.PortB.RxQueues(),
			txq:  pair.PortB.TxQueues(),
		})
		vnet.Ports = append(vnet.Ports, pair.PortA)
		ports = append(ports, pair.PortA.ID())
		bridgePorts = append(bridgePorts, pair.PortB.ID())
	}

	vnet.logger.Info("vnet ready",
		zap.Int("nNodes", cfg.NNodes),
		zap.Ints("ports", ports),
		zap.Ints("bridgePorts", bridgePorts),
	)
	return vnet, nil
}
