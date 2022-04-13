package l3

import (
	"github.com/usnistgov/ndn-dpdk/core/events"
)

const (
	evtStateChange = "StateChange"
)

// TransportBaseConfig contains parameters to NewTransportBase.
type TransportBaseConfig struct {
	MTU int
}

// TransportBase is an optional helper for implementing Transport.
type TransportBase struct {
	mtu     int
	state   TransportState
	emitter *events.Emitter
}

// MTU implements Transport interface.
func (b *TransportBase) MTU() int {
	return b.mtu
}

// State implements Transport interface.
func (b *TransportBase) State() TransportState {
	return b.state
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
	if p.b.state == st {
		return
	}
	p.b.state = st
	p.b.emitter.Emit(evtStateChange, st)
}

// NewTransportBase creates helpers for implementing Transport.
func NewTransportBase(cfg TransportBaseConfig) (b *TransportBase, p *TransportBasePriv) {
	b = &TransportBase{
		mtu:     cfg.MTU,
		state:   TransportUp,
		emitter: events.NewEmitter(),
	}
	p = &TransportBasePriv{
		b: b,
	}
	return
}
