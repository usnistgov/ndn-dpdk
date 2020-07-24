// Package sockettransport implements a transport based on stream or datagram sockets.
package sockettransport

import (
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/core/emission"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// Config contains socket transport configuration.
type Config struct {
	ndn.TransportQueueConfig

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
	cfg.RedialBackoffMaximum = time.Duration(math.MaxInt64(int64(cfg.RedialBackoffMaximum), int64(cfg.RedialBackoffInitial)))
}

// Transport is an ndn.Transport that communicates over a socket.
//
// A transport has automatic error handling: if a socket error occurs, the transport automatically
// redials the socket. In case the socket cannot be redialed, the transport remains in "down" status.
//
// A transport closes itself after its TX channel has been closed.
type Transport struct {
	cfg     Config
	impl    impl
	conn    atomic.Value
	rx      chan []byte
	tx      chan []byte
	err     chan error
	closing chan bool
	closed  int32 // atomic bool
	emitter *emission.Emitter

	// IsDown indicates whether the transport is down (socket is disconnected).
	IsDown bool

	// NRedial indicates how many times the socket has been redialed.
	NRedials int
}

// New creates a socket transport.
func New(conn net.Conn, cfg Config) (*Transport, error) {
	network := conn.LocalAddr().Network()
	impl, ok := implByNetwork[network]
	if !ok {
		return nil, fmt.Errorf("unknown network %s", network)
	}

	var tr Transport
	tr.cfg = cfg
	tr.cfg.applyDefaults()
	tr.impl = impl
	tr.conn.Store(conn)

	tr.rx = make(chan []byte, tr.cfg.RxQueueSize)
	tr.tx = make(chan []byte, tr.cfg.TxQueueSize)
	tr.err = make(chan error, 1) // 1-item buffer allows rxLoop to send its error after redialLoop exits
	tr.closing = make(chan bool)
	tr.emitter = emission.NewEmitter()
	go tr.rxLoop()
	go tr.txLoop()
	go tr.redialLoop()
	return &tr, nil
}

// Rx returns the RX channel.
func (tr *Transport) Rx() <-chan []byte {
	return tr.rx
}

// Tx returns the TX channel.
func (tr *Transport) Tx() chan<- []byte {
	return tr.tx
}

// Conn returns the underlying socket.
// Caller may gather information from this socket, but should not close or send/receive on it.
// The socket may be replaced during redialing.
func (tr *Transport) Conn() net.Conn {
	return tr.conn.Load().(net.Conn)
}

// OnStateChange registers a callback to be invoked when the transport goes up or down.
func (tr *Transport) OnStateChange(cb func(isDown bool)) io.Closer {
	return tr.emitter.On(eventStateChange, cb)
}

func (tr *Transport) isClosed() bool {
	return atomic.LoadInt32(&tr.closed) != 0
}

func (tr *Transport) rxLoop() {
	for !tr.isClosed() {
		e := tr.impl.RxLoop(tr)
		tr.err <- e
	}
	close(tr.rx)
}

func (tr *Transport) txLoop() {
	for {
		wire, ok := <-tr.tx
		if !ok {
			break
		}

		_, e := tr.Conn().Write(wire)
		if e != nil {
			tr.err <- e
		}
	}
	tr.closing <- true
	atomic.StoreInt32(&tr.closed, 1)
	tr.Conn().Close()
}

func (tr *Transport) redialLoop() {
	for {
		select {
		case <-tr.closing:
			tr.drainErrors()
			return
		case e := <-tr.err:
			tr.handleError(e)
		}
	}
}

func (tr *Transport) drainErrors() {
	for {
		select {
		case <-tr.err:
		default:
			return
		}
	}
}

func (tr *Transport) handleError(e error) {
	tr.setDown(true)

	backoff := tr.cfg.RedialBackoffInitial
	for !tr.isClosed() {
		time.Sleep(backoff)
		backoff = time.Duration(math.MinInt64(int64(backoff*2), int64(tr.cfg.RedialBackoffMaximum)))

		conn, e := tr.impl.Redial(tr.Conn())
		tr.NRedials++
		if e == nil {
			tr.conn.Store(conn)
			tr.drainErrors()
			tr.setDown(false)
			return
		}
	}
}

func (tr *Transport) setDown(isDown bool) {
	tr.IsDown = isDown
	tr.emitter.EmitSync(eventStateChange, isDown)
}

const (
	eventStateChange = "StateChange"
)
