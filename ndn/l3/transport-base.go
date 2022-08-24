package l3

import (
	"sync/atomic"

	"github.com/usnistgov/ndn-dpdk/core/events"
)

const (
	evtStateChange = "StateChange"
)

// TransportBaseConfig contains parameters to NewTransportBase.
type TransportBaseConfig struct {
	MTU          int
	InitialState TransportState
}

// TransportBase is an optional helper for implementing Transport.
type TransportBase struct {
	mtu     int
	state   atomic.Int32
	emitter *events.Emitter
}

// MTU implements Transport interface.
func (b *TransportBase) MTU() int {
	return b.mtu
}

// State implements Transport interface.
func (b *TransportBase) State() TransportState {
	return TransportState(b.state.Load())
}

// OnStateChange implements Transport interface.
func (b *TransportBase) OnStateChange(cb func(st TransportState)) (cancel func()) {
	return b.emitter.On(evtStateChange, cb)
}

// TransportBasePriv is an optional helper for implementing Transport interface.
type TransportBasePriv struct {
	b *TransportBase
}

// SetState changes transport state.
func (p *TransportBasePriv) SetState(st TransportState) {
	if p.b.state.Swap(int32(st)) == int32(st) {
		return
	}
	p.b.emitter.Emit(evtStateChange, st)
}

// NewTransportBase creates helpers for implementing Transport.
func NewTransportBase(cfg TransportBaseConfig) (b *TransportBase, p *TransportBasePriv) {
	b = &TransportBase{
		mtu:     cfg.MTU,
		emitter: events.NewEmitter(),
	}
	b.state.Store(int32(cfg.InitialState))
	p = &TransportBasePriv{
		b: b,
	}
	return
}
