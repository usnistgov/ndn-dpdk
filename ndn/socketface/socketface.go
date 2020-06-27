package socketface

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
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Config contains socket face configuration.
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

// SocketFace is an ndn.L3Face that communicates over a socket.
type SocketFace struct {
	cfg       Config
	impl      impl
	conn      atomic.Value
	rx        chan *ndn.Packet
	tx        chan *ndn.Packet
	emitter   *emission.Emitter
	closing   int32 // atomic bool
	redialing int32 // atomic bool
	closeTx   chan bool
	quitWg    sync.WaitGroup // wait until rxLoop and txLoop quits

	// IsDown indicates whether the face is down (socket is disconnected).
	IsDown bool

	// NRedial indicates how many times the socket has been redialed.
	NRedials int
}

// New creates a SocketFace.
func New(conn net.Conn, cfg Config) (*SocketFace, error) {
	network := conn.LocalAddr().Network()
	impl, ok := implByNetwork[network]
	if !ok {
		return nil, fmt.Errorf("unknown network %s", network)
	}

	var face SocketFace
	face.cfg = cfg
	face.cfg.applyDefaults()
	face.impl = impl
	face.conn.Store(conn)

	face.rx = make(chan *ndn.Packet, face.cfg.RxQueueSize)
	face.tx = make(chan *ndn.Packet, face.cfg.TxQueueSize)
	face.emitter = emission.NewEmitter()
	face.closeTx = make(chan bool, 1)
	face.quitWg.Add(2)
	go face.rxLoop()
	go face.txLoop()
	return &face, nil
}

// Close closes the face.
func (face *SocketFace) Close() error {
	atomic.StoreInt32(&face.closing, 1)
	face.closeTx <- true
	face.GetConn().Close() // ignore error
	face.quitWg.Wait()
	return nil
}

// GetRx returns the RX channel.
func (face *SocketFace) GetRx() <-chan *ndn.Packet {
	return face.rx
}

// GetTx returns the TX channel.
func (face *SocketFace) GetTx() chan<- *ndn.Packet {
	return face.tx
}

// GetConn returns the underlying socket.
// Caller may gather information from this socket, but should not send/receive/close it.
// The socket may be replaced during redialing.
func (face *SocketFace) GetConn() net.Conn {
	return face.conn.Load().(net.Conn)
}

// OnStateChange registers a callback to be invoked when the face goes up or down.
func (face *SocketFace) OnStateChange(cb func(isDown bool)) io.Closer {
	return face.emitter.On(eventStateChange, cb)
}

func (face *SocketFace) rxLoop() {
	face.impl.RxLoop(face)
	close(face.rx)
	face.quitWg.Done()
}

func (face *SocketFace) txLoop() {
	for {
		select {
		case <-face.closeTx:
			face.quitWg.Done()
			return
		case packet := <-face.tx:
			wire, e := tlv.Encode(packet)
			if e != nil { // ignore encoding error
				continue
			}

			_, e = face.GetConn().Write(wire)
			if e != nil && face.handleError(e) { // handle socket error
				break
			}
		}
	}
}

func (face *SocketFace) handleError(e error) (stop bool) {
	if atomic.LoadInt32(&face.closing) != 0 {
		return true
	}

	var netErr net.Error
	if errors.As(e, &netErr) && netErr.Temporary() {
		return false
	}

	if atomic.CompareAndSwapInt32(&face.redialing, 0, 1) {
		defer atomic.StoreInt32(&face.redialing, 0)
		face.IsDown = true
		face.emitter.EmitSync(eventStateChange, true)

		backoff := face.cfg.RedialBackoffInitial
		for atomic.LoadInt32(&face.closing) == 0 {
			time.Sleep(backoff)
			backoff *= 2
			if backoff > face.cfg.RedialBackoffMaximum {
				backoff = face.cfg.RedialBackoffMaximum
			}

			conn, e := face.impl.Redial(face.GetConn())
			face.NRedials++
			if e == nil {
				face.conn.Store(conn)
				face.IsDown = false
				face.emitter.EmitSync(eventStateChange, false)
				break
			}
		}
	} else { // another goroutine is redialing
		for atomic.LoadInt32(&face.redialing) != 0 {
			runtime.Gosched()
		}
	}
	return atomic.LoadInt32(&face.closing) != 0
}

const (
	eventStateChange = "StateChange"
)
