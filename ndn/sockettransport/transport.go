// Package sockettransport implements a transport based on stream or datagram sockets.
package sockettransport

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
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
		impl:    impl,
		conn:    conn,
		backoff: retry.WithCappedDuration(cfg.RedialBackoffMaximum, retry.NewExponential(cfg.RedialBackoffInitial)),
	}
	tr.TransportBase, tr.p = l3.NewTransportBase(l3.TransportBaseConfig{
		MTU: cfg.MTU,
	})
	tr.ctx, tr.cancel = context.WithCancel(context.Background())
	return tr, nil
}

type transport struct {
	*l3.TransportBase
	p          *l3.TransportBasePriv
	impl       impl
	nRedials   atomic.Int32
	redialLock sync.Mutex
	conn       net.Conn
	rxBuffer   any
	backoff    retry.Backoff
	ctx        context.Context
	cancel     context.CancelFunc
}

func (tr *transport) Conn() net.Conn {
	return tr.conn
}

func (tr *transport) Counters() (cnt Counters) {
	cnt.NRedials = int(tr.nRedials.Load())
	return
}

func (tr *transport) Read(buf []byte) (n int, e error) {
	return tr.doRW(func() (n int, e error) {
		return tr.impl.Read(tr, buf)
	})
}

func (tr *transport) Write(buf []byte) (n int, e error) {
	return tr.doRW(func() (n int, e error) {
		return tr.conn.Write(buf)
	})
}

func (tr *transport) Close() error {
	tr.cancel()
	tr.p.SetState(l3.TransportClosed)
	tr.conn.Close()
	return nil
}

func (tr *transport) doRW(f func() (n int, e error)) (n int, e error) {
	if tr.ctx.Err() != nil {
		return 0, io.ErrClosedPipe
	}

	nRedialsEnter := tr.nRedials.Load()
	if n, e = f(); e == nil {
		return n, e
	}

	tr.redialLock.Lock()
	defer tr.redialLock.Unlock()
	if !tr.nRedials.CAS(nRedialsEnter, nRedialsEnter+1) { // another goroutine performed redial
		return 0, nil
	}

	e = retry.Do(tr.ctx, tr.backoff, func(ctx context.Context) error {
		tr.p.SetState(l3.TransportDown)
		conn, e := tr.impl.Redial(tr.conn)
		if e != nil {
			return retry.RetryableError(e)
		}

		tr.conn, tr.rxBuffer = conn, nil
		tr.p.SetState(l3.TransportUp)
		return nil
	})
	if e != nil { // transport closed
		return 0, io.ErrClosedPipe
	}
	return 0, nil
}
