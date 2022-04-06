package ndntestenv

import (
	"math/rand"
	"sync"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/zyedidia/generic"
)

// BridgeConfig contains Bridge parameters.
type BridgeConfig struct {
	FwA     l3.Forwarder
	FwB     l3.Forwarder
	RelayAB BridgeRelayConfig
	RelayBA BridgeRelayConfig
}

func (cfg *BridgeConfig) applyDefaults() {
	if cfg.FwA == nil {
		cfg.FwA = l3.NewForwarder()
	}
	if cfg.FwB == nil {
		cfg.FwB = l3.NewForwarder()
	}
	cfg.RelayAB.applyDefaults()
	cfg.RelayBA.applyDefaults()
}

// BridgeRelayConfig contains Bridge link loss.
type BridgeRelayConfig struct {
	Loss     float64 // between 0.0 and 1.0
	MinDelay time.Duration
	MaxDelay time.Duration
}

func (cfg *BridgeRelayConfig) applyDefaults() {
	cfg.Loss = generic.Clamp(cfg.Loss, 0.0, 1.0)
	if cfg.MinDelay < cfg.MaxDelay {
		cfg.MinDelay, cfg.MaxDelay = cfg.MaxDelay, cfg.MinDelay
	}
}

// Bridge links two l3.Forwarder and emulates a lossy link.
type Bridge struct {
	FwA, FwB     l3.Forwarder
	FaceA, FaceB l3.FwFace
	closing      chan struct{}
}

// Close detaches the link from forwarders.
func (br *Bridge) Close() error {
	close(br.closing)
	return nil
}

// NewBridge creates a Bridge.
func NewBridge(cfg BridgeConfig) (br *Bridge) {
	cfg.applyDefaults()
	br = &Bridge{
		FwA:     cfg.FwA,
		FwB:     cfg.FwB,
		closing: make(chan struct{}),
	}
	trA, trB := newBridgeTransport(), newBridgeTransport()
	trA.peer, trB.peer = trB, trA
	go trA.loop(cfg.RelayAB, br.closing)
	go trB.loop(cfg.RelayBA, br.closing)
	faceA, _ := l3.NewFace(trA, l3.FaceConfig{})
	faceB, _ := l3.NewFace(trB, l3.FaceConfig{})
	br.FaceA, _ = br.FwA.AddFace(faceA)
	br.FaceB, _ = br.FwB.AddFace(faceB)
	br.FaceA.AddRoute(ndn.Name{})
	br.FaceB.AddRoute(ndn.Name{})
	return br
}

type bridgeTransport struct {
	*l3.TransportBase
	p    *l3.TransportBasePriv
	peer *bridgeTransport
}

func (tr *bridgeTransport) loop(relay BridgeRelayConfig, closing <-chan struct{}) {
	delay := func() time.Duration { return relay.MinDelay }
	if relay.MinDelay != relay.MaxDelay {
		delayRange := float64(relay.MaxDelay - relay.MinDelay)
		delay = func() time.Duration { return relay.MinDelay + time.Duration(delayRange*rand.Float64()) }
	}

	var wg sync.WaitGroup
	defer func() {
		wg.Wait()
		close(tr.peer.p.Rx)
	}()

	for {
		select {
		case <-closing:
			return
		case pkt, ok := <-tr.p.Tx:
			if !ok {
				return
			}
			if rand.Float64() < relay.Loss {
				continue
			}

			wg.Add(1)
			go func(delay time.Duration, pkt []byte) {
				time.Sleep(delay)
				tr.peer.p.Rx <- pkt
			}(delay(), pkt)
		}
	}
}

func newBridgeTransport() (tr *bridgeTransport) {
	tr = &bridgeTransport{}
	tr.TransportBase, tr.p = l3.NewTransportBase(l3.TransportBaseConfig{})
	return tr
}
