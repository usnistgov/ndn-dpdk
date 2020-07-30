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
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

// Config contains socket transport configuration.
type Config struct {
	l3.TransportQueueConfig

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

// Counters contains socket transport counters.
type Counters struct {
	// NRedials indicates how many times the socket has been redialed.
	NRedials int `json:"nRedials"`

	// RxQueueLength is the current number of packets in the RX queue.
	RxQueueLength int

	// RxQueueLength is the current number of packets in the TX queue.
	TxQueueLength int
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

	// IsDown returns whether the transport is down (socket is disconnected).
	IsDown() bool

	// OnStateChange registers a callback to be invoked when the transport goes up or down.
	OnStateChange(cb func(isDown bool)) io.Closer

	// Counters returns current counters.
	Counters() Counters
}

type transport struct {
	cfg     Config
	impl    impl
	conn    atomic.Value // net.Conn
	rx      chan []byte
	tx      chan []byte
	err     chan error
	cnt     Counters
	isDown  bool
	closing chan bool
	closed  int32 // atomic bool
	emitter *emission.Emitter
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
		rx:      make(chan []byte, cfg.RxQueueSize),
		tx:      make(chan []byte, cfg.TxQueueSize),
		err:     make(chan error, 1), // 1-item buffer allows rxLoop to send its error after redialLoop exits
		closing: make(chan bool),
		emitter: emission.NewEmitter(),
	}

	tr.conn.Store(conn)
	go tr.rxLoop()
	go tr.txLoop()
	go tr.redialLoop()
	return tr, nil
}

func (tr *transport) Rx() <-chan []byte {
	return tr.rx
}

func (tr *transport) Tx() chan<- []byte {
	return tr.tx
}

func (tr *transport) Conn() net.Conn {
	return tr.conn.Load().(net.Conn)
}

func (tr *transport) IsDown() bool {
	return tr.isDown
}

func (tr *transport) OnStateChange(cb func(isDown bool)) io.Closer {
	return tr.emitter.On(eventStateChange, cb)
}

func (tr *transport) Counters() (cnt Counters) {
	cnt = tr.cnt
	cnt.RxQueueLength = len(tr.rx)
	cnt.TxQueueLength = len(tr.tx)
	return cnt
}

func (tr *transport) isClosed() bool {
	return atomic.LoadInt32(&tr.closed) != 0
}

func (tr *transport) rxLoop() {
	for !tr.isClosed() {
		e := tr.impl.RxLoop(tr)
		tr.err <- e
	}
	close(tr.rx)
}

func (tr *transport) txLoop() {
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

func (tr *transport) redialLoop() {
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
		backoff = time.Duration(math.MinInt64(int64(backoff*2), int64(tr.cfg.RedialBackoffMaximum)))

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
	tr.isDown = isDown
	tr.emitter.EmitSync(eventStateChange, isDown)
}

const (
	eventStateChange = "StateChange"
)
