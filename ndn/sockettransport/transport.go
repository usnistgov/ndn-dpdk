// Package sockettransport implements a transport based on stream or datagram sockets.
package sockettransport

import (
	"fmt"
	"net"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/zyedidia/generic"
	"go.uber.org/atomic"
)

// Config contains socket transport configuration.
type Config struct {
	l3.TransportQueueConfig

	// MTU is maximum outgoing packet size.
	// The default is 0, which means unlimited.
	MTU int

	// RxBufferLength is the packet buffer length allocated for incoming packets.
	// The default is 16384.
	// Packet larger than this length cannot be received.
	RxBufferLength int

	// RedialBackoffInitial is the initial backoff period during redialing.
	// The default is 100ms.
	RedialBackoffInitial time.Duration

	// RedialBackoffMaximum is the maximum backoff period during redialing.
	// The default is 60s.
	// The minimum is RedialBackoffInitial.
	RedialBackoffMaximum time.Duration
}

func (cfg *Config) applyDefaults() {
	cfg.ApplyTransportQueueConfigDefaults()

	if cfg.RxBufferLength <= 0 {
		cfg.RxBufferLength = 16384
	}
	if cfg.RedialBackoffInitial <= 0 {
		cfg.RedialBackoffInitial = 100 * time.Millisecond
	}
	if cfg.RedialBackoffMaximum <= 0 {
		cfg.RedialBackoffMaximum = 60 * time.Second
	}
	cfg.RedialBackoffMaximum = generic.Max(cfg.RedialBackoffMaximum, cfg.RedialBackoffInitial)
}

// Counters contains socket transport counters.
type Counters struct {
	// NRedials indicates how many times the socket has been redialed.
	NRedials int `json:"nRedials"`

	// RxQueueLength is the current number of packets in the RX queue.
	RxQueueLength int `json:"rxQueueLength"`

	// TxQueueLength is the current number of packets in the TX queue.
	TxQueueLength int `json:"txQueueLength"`
}

func (cnt Counters) String() string {
	return fmt.Sprintf("%dredials, rx %dqueued, tx %dqueued", cnt.NRedials, cnt.RxQueueLength, cnt.TxQueueLength)
}

// Transport is an l3.Transport that communicates over a socket.
//
// A transport has automatic error handling: if a socket error occurs, the transport automatically
// redials the socket. In case the socket cannot be redialed, the transport remains in "down" status.
//
// A transport closes itself after its TX channel has been closed.
type Transport interface {
	l3.Transport

	// Conn returns the underlying socket.
	// Caller may gather information from this socket, but should not close or send/receive on it.
	// The socket may be replaced during redialing.
	Conn() net.Conn

	// Counters returns current counters.
	Counters() Counters
}

// New creates a socket transport.
func New(conn net.Conn, cfg Config) (Transport, error) {
	network := conn.LocalAddr().Network()
	impl, ok := implByNetwork[network]
	if !ok {
		return nil, fmt.Errorf("unknown network %s", network)
	}
	cfg.applyDefaults()

	tr := &transport{
		cfg:     cfg,
		impl:    impl,
		err:     make(chan error, 1), // 1-item buffer allows rxLoop to send its error after redialLoop exits
		closing: make(chan struct{}),
	}
	tr.TransportBase, tr.p = l3.NewTransportBase(l3.TransportBaseConfig{
		TransportQueueConfig: cfg.TransportQueueConfig,
		MTU:                  cfg.MTU,
	})

	tr.conn.Store(conn)
	go tr.rxLoop()
	go tr.txLoop()
	go tr.redialLoop()
	return tr, nil
}

type transport struct {
	*l3.TransportBase
	p       *l3.TransportBasePriv
	cfg     Config
	impl    impl
	conn    atomic.Value // net.Conn
	err     chan error
	cnt     Counters
	closing chan struct{}
	closed  atomic.Bool
}

func (tr *transport) Conn() net.Conn {
	return tr.conn.Load().(net.Conn)
}

func (tr *transport) Counters() (cnt Counters) {
	cnt = tr.cnt
	cnt.RxQueueLength = len(tr.p.Rx)
	cnt.TxQueueLength = len(tr.p.Tx)
	return cnt
}

func (tr *transport) isClosed() bool {
	return tr.closed.Load()
}

func (tr *transport) rxLoop() {
	for !tr.isClosed() {
		e := tr.impl.RxLoop(tr)
		tr.err <- e
	}
	close(tr.p.Rx)
	tr.p.SetState(l3.TransportClosed)
}

func (tr *transport) txLoop() {
	for {
		wire, ok := <-tr.p.Tx
		if !ok {
			break
		}

		_, e := tr.Conn().Write(wire)
		if e != nil {
			tr.err <- e
		}
	}
	close(tr.closing)
}

func (tr *transport) redialLoop() {
CLOSING:
	for {
		select {
		case <-tr.closing:
			break CLOSING
		case e := <-tr.err:
			tr.handleError(e)
		}
	}

	tr.closed.Store(true)
	tr.drainErrors()
	tr.Conn().Close()
}

func (tr *transport) drainErrors() {
	for {
		select {
		case <-tr.err:
		default:
			return
		}
	}
}

func (tr *transport) handleError(e error) {
	tr.setDown(true)

	backoff := tr.cfg.RedialBackoffInitial
	for !tr.isClosed() {
		time.Sleep(backoff)
		backoff = generic.Min(backoff*2, tr.cfg.RedialBackoffMaximum)

		conn, e := tr.impl.Redial(tr.Conn())
		tr.cnt.NRedials++
		if e == nil {
			tr.conn.Store(conn)
			tr.drainErrors()
			tr.setDown(false)
			return
		}
	}
}

func (tr *transport) setDown(isDown bool) {
	st := l3.TransportUp
	if isDown {
		st = l3.TransportDown
	}
	tr.p.SetState(st)
}
