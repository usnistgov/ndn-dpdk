//go:build linux

package memiftransport

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/FDio/vpp/extras/gomemif/memif"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

var handleCoexist = NewCoexistMap()

func (loc *Locator) toArguments() (a *memif.Arguments, e error) {
	if e := loc.Validate(); e != nil {
		return nil, e
	}
	loc.ApplyDefaults(RoleClient)

	return &memif.Arguments{
		Id:       uint32(loc.ID),
		IsMaster: loc.Role == RoleServer,
		Name:     os.Args[0],
		MemoryConfig: memif.MemoryConfig{
			NumQueuePairs:    1,
			Log2RingSize:     loc.rsize(),
			PacketBufferSize: uint32(loc.Dataroom),
		},
	}, nil
}

func newHandle(loc Locator, setState func(l3.TransportState)) (hdl *handle, e error) {
	a, e := loc.toArguments()
	if e != nil {
		return nil, e
	}
	if e := handleCoexist.Check(loc); e != nil {
		return nil, e
	}

	sock, e := memif.NewSocket(os.Args[0], loc.SocketName)
	if e != nil {
		return nil, fmt.Errorf("memif.NewSocket %w", e)
	}

	if setState == nil {
		setState = func(l3.TransportState) {}
	}
	hdl = &handle{
		Locator:    loc,
		sock:       sock,
		memifError: make(chan error),
		setState:   setState,
	}

	a.ConnectedFunc = hdl.memifConnected
	a.DisconnectedFunc = hdl.memifDisconnected
	hdl.intf, e = sock.NewInterface(a)
	if e != nil {
		sock.Delete()
		return nil, fmt.Errorf("sock.NewInterface %w", e)
	}

	handleCoexist.Add(loc)
	hdl.sock.StartPolling(hdl.memifError)
	if !hdl.intf.IsMaster() {
		hdl.intf.RequestConnection()
	}
	return hdl, nil
}

type handle struct {
	Locator Locator

	sock       *memif.Socket
	memifError chan error
	intf       *memif.Interface
	setState   func(l3.TransportState)

	mutex  sync.RWMutex
	rxq    *memif.Queue
	txq    *memif.Queue
	closed bool
}

var _ io.ReadWriteCloser = &handle{}

func (hdl *handle) memifConnected(intf *memif.Interface) error {
	hdl.mutex.Lock()
	defer hdl.mutex.Unlock()
	hdl.rxq, _ = intf.GetRxQueue(0)
	hdl.txq, _ = intf.GetTxQueue(0)
	hdl.setState(l3.TransportUp)
	return nil
}

func (hdl *handle) memifDisconnected(intf *memif.Interface) error {
	hdl.mutex.Lock()
	defer hdl.mutex.Unlock()
	hdl.rxq = nil
	hdl.txq = nil
	hdl.setState(l3.TransportDown)
	return nil
}

func (hdl *handle) Read(buf []byte) (n int, e error) {
	hdl.mutex.RLock()
	defer hdl.mutex.RUnlock()

	if hdl.closed {
		return 0, io.EOF
	}

	if hdl.rxq != nil {
		n, e = hdl.rxq.ReadPacket(buf)
	}

	if e == nil {
		if n == 0 {
			runtime.Gosched()
			e = io.ErrNoProgress
		} else if n > len(buf) {
			e = io.ErrShortBuffer
		}
	}
	return n, e
}

func (hdl *handle) Write(buf []byte) (n int, e error) {
	hdl.mutex.RLock()
	defer hdl.mutex.RUnlock()

	if hdl.txq != nil {
		n = hdl.txq.WritePacket(buf)
	}

	if n < len(buf) {
		return n, io.ErrShortWrite
	}
	return n, nil
}

func (hdl *handle) Close() error {
	hdl.mutex.Lock()
	hdl.closed = true
	hdl.setState(l3.TransportClosed)
	hdl.mutex.Unlock()

	hdl.sock.Delete()
	handleCoexist.Remove(hdl.Locator)
	return nil
}
