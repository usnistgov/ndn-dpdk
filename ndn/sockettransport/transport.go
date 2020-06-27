package sockettransport

import (
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/emission"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Config contains socket transport configuration.
type Config struct {
	// RxBufferLength is the packet buffer length allocated for incoming packets.
	// The default is 16384.
	// Packet larger than this length cannot be received.
	RxBufferLength int

	// RxChanBuffer is the Go channel buffer size of RX channel.
	// The default is 64.
	RxQueueSize int

	// TxChanBuffer is the Go channel buffer size of TX channel.
	// The default is 64.
	TxQueueSize int

	// RedialBackoffInitial is the initial backoff period during redialing.
	// The default is 100ms.
	RedialBackoffInitial time.Duration

	// RedialBackoffMaximum is the maximum backoff period during redialing.
	// The default is 60s.
	// The minimum is RedialBackoffInitial.
	RedialBackoffMaximum time.Duration
}

func (cfg *Config) applyDefaults() {
	if cfg.RxBufferLength <= 0 {
		cfg.RxBufferLength = 16384
	}
	if cfg.RxQueueSize <= 0 {
		cfg.RxQueueSize = 64
	}
	if cfg.TxQueueSize <= 0 {
		cfg.TxQueueSize = 64
	}
	if cfg.RedialBackoffInitial <= 0 {
		cfg.RedialBackoffInitial = 100 * time.Millisecond
	}
	if cfg.RedialBackoffMaximum <= 0 {
		cfg.RedialBackoffMaximum = 60 * time.Second
	}
	if cfg.RedialBackoffMaximum < cfg.RedialBackoffInitial {
		cfg.RedialBackoffMaximum = cfg.RedialBackoffInitial
	}
}

// Transport is an ndn.Transport that communicates over a socket.
type Transport struct {
	cfg       Config
	impl      impl
	conn      atomic.Value
	rx        chan []byte
	tx        chan []byte
	emitter   *emission.Emitter
	closing   int32 // atomic bool
	redialing int32 // atomic bool
	closeTx   chan bool
	quitWg    sync.WaitGroup // wait until rxLoop and txLoop quits

	// IsDown indicates whether the transport is down (socket is disconnected).
	IsDown bool

	// NRedial indicates how many times the socket has been redialed.
	NRedials int
}

// New creates a Transport.
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
	tr.emitter = emission.NewEmitter()
	tr.closeTx = make(chan bool, 1)
	tr.quitWg.Add(2)
	go tr.rxLoop()
	go tr.txLoop()
	return &tr, nil
}

// Close closes the tr.
func (tr *Transport) Close() error {
	atomic.StoreInt32(&tr.closing, 1)
	tr.closeTx <- true
	tr.GetConn().Close() // ignore error
	tr.quitWg.Wait()
	return nil
}

// GetRx returns the RX channel.
func (tr *Transport) GetRx() <-chan []byte {
	return tr.rx
}

// GetTx returns the TX channel.
func (tr *Transport) GetTx() chan<- []byte {
	return tr.tx
}

// GetConn returns the underlying socket.
// Caller may gather information from this socket, but should not close or send/receive on it.
// The socket may be replaced during redialing.
func (tr *Transport) GetConn() net.Conn {
	return tr.conn.Load().(net.Conn)
}

// OnStateChange registers a callback to be invoked when the transport goes up or down.
func (tr *Transport) OnStateChange(cb func(isDown bool)) io.Closer {
	return tr.emitter.On(eventStateChange, cb)
}

func (tr *Transport) rxLoop() {
	tr.impl.RxLoop(tr)
	close(tr.rx)
	tr.quitWg.Done()
}

func (tr *Transport) txLoop() {
	for {
		select {
		case <-tr.closeTx:
			tr.quitWg.Done()
			return
		case packet := <-tr.tx:
			wire, e := tlv.Encode(packet)
			if e != nil { // ignore encoding error
				continue
			}

			_, e = tr.GetConn().Write(wire)
			if e != nil && tr.handleError(e) { // handle socket error
				break
			}
		}
	}
}

func (tr *Transport) handleError(e error) (stop bool) {
	if atomic.LoadInt32(&tr.closing) != 0 {
		return true
	}

	var netErr net.Error
	if errors.As(e, &netErr) && netErr.Temporary() {
		return false
	}

	if atomic.CompareAndSwapInt32(&tr.redialing, 0, 1) {
		defer atomic.StoreInt32(&tr.redialing, 0)
		tr.IsDown = true
		tr.emitter.EmitSync(eventStateChange, true)

		backoff := tr.cfg.RedialBackoffInitial
		for atomic.LoadInt32(&tr.closing) == 0 {
			time.Sleep(backoff)
			backoff *= 2
			if backoff > tr.cfg.RedialBackoffMaximum {
				backoff = tr.cfg.RedialBackoffMaximum
			}

			conn, e := tr.impl.Redial(tr.GetConn())
			tr.NRedials++
			if e == nil {
				tr.conn.Store(conn)
				tr.IsDown = false
				tr.emitter.EmitSync(eventStateChange, false)
				break
			}
		}
	} else { // another goroutine is redialing
		for atomic.LoadInt32(&tr.redialing) != 0 {
			runtime.Gosched()
		}
	}
	return atomic.LoadInt32(&tr.closing) != 0
}

const (
	eventStateChange = "StateChange"
)
