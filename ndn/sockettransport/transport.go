// Package sockettransport implements a transport based on stream or datagram sockets.
package sockettransport

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/sethvargo/go-retry"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/zyedidia/generic"
	"go.uber.org/atomic"
)

// Config contains socket transport configuration.
type Config struct {
	// MTU is maximum outgoing packet size.
	// The default is 16384.
	MTU int

	// RedialBackoffInitial is the initial backoff period during redialing.
	// The default is 100ms.
	RedialBackoffInitial time.Duration

	// RedialBackoffMaximum is the maximum backoff period during redialing.
	// The default is 60s.
	// The minimum is RedialBackoffInitial.
	RedialBackoffMaximum time.Duration
}

func (cfg *Config) applyDefaults() {
	if cfg.MTU <= 0 {
		cfg.MTU = 16384
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
}

func (cnt Counters) String() string {
	return fmt.Sprintf("%dredials", cnt.NRedials)
}

// Transport is an l3.Transport that communicates over a socket.
//
// A transport has automatic error handling: if a socket error occurs, the transport automatically
// redials the socket. In case the socket cannot be redialed, the transport remains in "down" status.
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
		impl: impl,
		err:  make(chan error, 1), // 1-item buffer allows rxLoop to send its error after redialLoop exits
		// closing: make(chan struct{}),
		backoff: retry.WithCappedDuration(cfg.RedialBackoffMaximum, retry.NewExponential(cfg.RedialBackoffInitial)),
	}
	tr.TransportBase, tr.p = l3.NewTransportBase(l3.TransportBaseConfig{
		MTU: cfg.MTU,
	})
	tr.ctx, tr.cancel = context.WithCancel(context.Background())

	tr.setConn(conn)
	go tr.redialLoop()
	return tr, nil
}

type trConn struct {
	conn net.Conn
	rx   any
}

type transport struct {
	*l3.TransportBase
	p       *l3.TransportBasePriv
	impl    impl
	conn    atomic.Value // *trConn
	cnt     Counters
	err     chan error
	backoff retry.Backoff
	ctx     context.Context
	cancel  context.CancelFunc
}

func (tr *transport) Conn() net.Conn {
	if tr.ctx.Err() != nil {
		return nil
	}
	return tr.conn.Load().(*trConn).conn
}

func (tr *transport) setConn(conn net.Conn) {
	tr.conn.Store(&trConn{conn, nil})
}

func (tr *transport) Counters() (cnt Counters) {
	return tr.cnt
}

func (tr *transport) Read(buf []byte) (n int, e error) {
	if tr.ctx.Err() != nil {
		return 0, io.ErrClosedPipe
	}

	trc := tr.conn.Load().(*trConn)
	n, e = tr.impl.Read(tr, trc, buf)
	if e != nil {
		tr.err <- e
	}
	return n, nil
}

func (tr *transport) Write(buf []byte) (n int, e error) {
	conn := tr.Conn()
	if conn == nil {
		return 0, io.ErrClosedPipe
	}

	n, e = conn.Write(buf)
	if e != nil {
		tr.err <- e
	}
	return n, nil
}

func (tr *transport) Close() error {
	tr.cancel()
	return nil
}

func (tr *transport) redialLoop() {
CLOSING:
	for {
		select {
		case <-tr.ctx.Done():
			break CLOSING
		case e := <-tr.err:
			tr.handleError(e)
		}
	}

	tr.p.SetState(l3.TransportClosed)
	tr.conn.Load().(*trConn).conn.Close()
	tr.drainErrors()
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
	tr.p.SetState(l3.TransportDown)

	retry.Do(tr.ctx, tr.backoff, func(ctx context.Context) error {
		conn, e := tr.impl.Redial(tr.Conn())
		tr.cnt.NRedials++
		if e == nil {
			tr.setConn(conn)
			tr.drainErrors()
			tr.p.SetState(l3.TransportUp)
			return nil
		}
		return retry.RetryableError(e)
	})
}
