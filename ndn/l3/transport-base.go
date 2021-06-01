package l3

import (
	"github.com/usnistgov/ndn-dpdk/core/events"
)

const (
	evtStateChange = "StateChange"
)

// TransportBase is an optional helper for implementing Transport.
type TransportBase struct {
	rx      <-chan []byte
	tx      chan<- []byte
	state   TransportState
	emitter *events.Emitter
}

// Rx implements Transport interface.
func (b *TransportBase) Rx() <-chan []byte {
	return b.rx
}

// Tx implements Transport interface.
func (b *TransportBase) Tx() chan<- []byte {
	return b.tx
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
	b  *TransportBase
	Rx chan<- []byte
	Tx <-chan []byte
}

// SetState changes transport state.
func (p *TransportBasePriv) SetState(st TransportState) {
	if p.b.state == st {
		return
	}
	p.b.state = st
	p.b.emitter.EmitSync(evtStateChange, st)
}

// NewTransportBase creates helpers for implementing Transport.
func NewTransportBase(qcfg TransportQueueConfig) (b *TransportBase, p *TransportBasePriv) {
	qcfg.ApplyTransportQueueConfigDefaults()
	rx := make(chan []byte, qcfg.RxQueueSize)
	tx := make(chan []byte, qcfg.TxQueueSize)
	b = &TransportBase{
		rx:      rx,
		tx:      tx,
		state:   TransportUp,
		emitter: events.NewEmitter(),
	}
	p = &TransportBasePriv{
		b:  b,
		Rx: rx,
		Tx: tx,
	}
	return
}
