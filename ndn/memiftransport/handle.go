package memiftransport

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/FDio/vpp/extras/gomemif/memif"
	"github.com/google/gopacket"
	"github.com/pkg/math"
)

// Role indicates memif role.
type Role int

// Role constants.
const (
	RoleServer Role = iota
	RoleClient
)

func newHandle(loc Locator, role Role) (hdl *handle, e error) {
	sock, e := memif.NewSocket(os.Args[0], loc.SocketName)
	if e != nil {
		return nil, fmt.Errorf("memif.NewSocket %w", e)
	}

	hdl = &handle{
		sock:       sock,
		memifError: make(chan error),
		dataroom:   loc.Dataroom,
	}

	a := &memif.Arguments{
		IsMaster:         role == RoleServer,
		ConnectedFunc:    hdl.memifConnected,
		DisconnectedFunc: hdl.memifDisconnected,
	}
	loc.toArguments(a)
	hdl.intf, e = sock.NewInterface(a)
	if e != nil {
		sock.Delete()
		return nil, fmt.Errorf("sock.NewInterface %w", e)
	}

	hdl.sock.StartPolling(hdl.memifError)
	if !hdl.intf.IsMaster() {
		hdl.intf.RequestConnection()
	}
	return hdl, nil
}

type handle struct {
	sock       *memif.Socket
	memifError chan error
	intf       *memif.Interface
	dataroom   int

	mutex  sync.RWMutex
	rxq    *memif.Queue
	txq    *memif.Queue
	closed error
}

func (hdl *handle) memifConnected(intf *memif.Interface) error {
	hdl.mutex.Lock()
	defer hdl.mutex.Unlock()
	hdl.rxq, _ = intf.GetRxQueue(0)
	hdl.txq, _ = intf.GetTxQueue(0)
	return nil
}

func (hdl *handle) memifDisconnected(intf *memif.Interface) error {
	hdl.mutex.Lock()
	defer hdl.mutex.Unlock()
	hdl.rxq = nil
	hdl.txq = nil
	return nil
}

func (hdl *handle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, e error) {
	data = make([]byte, hdl.dataroom)
	for {
		ci, e = hdl.recvPacket(data)
		if e != nil || ci.CaptureLength > 0 {
			data = data[:ci.CaptureLength]
			return
		}
		runtime.Gosched()
	}
}

func (hdl *handle) recvPacket(buf []byte) (ci gopacket.CaptureInfo, e error) {
	hdl.mutex.RLock()
	defer hdl.mutex.RUnlock()

	if hdl.closed != nil {
		e = hdl.closed
	} else if hdl.rxq != nil {
		ci.Length, e = hdl.rxq.ReadPacket(buf)
		ci.CaptureLength = math.MinInt(ci.Length, hdl.dataroom)
	}
	return
}

func (hdl *handle) WritePacketData(pkt []byte) error {
	hdl.mutex.RLock()
	defer hdl.mutex.RUnlock()

	if hdl.txq != nil {
		hdl.txq.WritePacket(pkt)
	}
	return nil
}

func (hdl *handle) Close() error {
	func() {
		hdl.mutex.Lock()
		defer hdl.mutex.Unlock()
		hdl.closed = io.EOF
	}()

	hdl.sock.Delete()
	return nil
}
