package ndntestenv

import (
	"math/rand/v2"
	"net"
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

func (cfg BridgeRelayConfig) makeLoss() func() (loss bool) {
	if cfg.Loss == 0 {
		return func() (loss bool) { return false }
	}
	return func() (loss bool) { return rand.Float64() < cfg.Loss }
}

func (cfg BridgeRelayConfig) makeDelay() func() (delay time.Duration) {
	if cfg.MinDelay == cfg.MaxDelay {
		return func() (delay time.Duration) { return cfg.MinDelay }
	}

	delayRange := float64(cfg.MaxDelay - cfg.MinDelay)
	return func() time.Duration { return cfg.MinDelay + time.Duration(delayRange*rand.Float64()) }
}

// Bridge links two l3.Forwarder and emulates a lossy link.
type Bridge struct {
	FwA, FwB     l3.Forwarder
	FaceA, FaceB l3.FwFace
	trA, trB     *bridgeTransport
}

// Close detaches the link from forwarders.
func (br *Bridge) Close() error {
	br.trA.Close()
	br.trB.Close()
	return nil
}

// NewBridge creates a Bridge.
func NewBridge(cfg BridgeConfig) (br *Bridge) {
	cfg.applyDefaults()
	br = &Bridge{
		FwA: cfg.FwA,
		FwB: cfg.FwB,
	}
	connA, connB := net.Pipe()
	br.trA, br.trB = newBridgeTransport(connA, cfg.RelayAB), newBridgeTransport(connB, cfg.RelayBA)
	faceA, _ := l3.NewFace(br.trA, l3.FaceConfig{})
	faceB, _ := l3.NewFace(br.trB, l3.FaceConfig{})
	br.FaceA, _ = br.FwA.AddFace(faceA)
	br.FaceB, _ = br.FwB.AddFace(faceB)
	br.FaceA.AddRoute(ndn.Name{})
	br.FaceB.AddRoute(ndn.Name{})
	return br
}

type bridgeTransport struct {
	net.Conn
	*l3.TransportBase
	p *l3.TransportBasePriv

	loss  func() (loss bool)
	delay func() (delay time.Duration)
}

func (tr *bridgeTransport) Write(buf []byte) (n int, e error) {
	if !tr.loss() {
		go func(delay time.Duration, pkt []byte) {
			time.Sleep(delay)
			tr.Conn.Write(pkt)
		}(tr.delay(), buf)
	}
	return len(buf), nil
}

func newBridgeTransport(conn net.Conn, relay BridgeRelayConfig) (tr *bridgeTransport) {
	tr = &bridgeTransport{
		Conn:  conn,
		loss:  relay.makeLoss(),
		delay: relay.makeDelay(),
	}
	tr.TransportBase, tr.p = l3.NewTransportBase(l3.TransportBaseConfig{
		MTU: 9000,
	})
	return tr
}
