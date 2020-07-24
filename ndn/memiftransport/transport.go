// Package memiftransport implements a transport over a shared memory packet interface (memif).
package memiftransport

import (
	"fmt"
	"os"
	"runtime"

	"github.com/FDio/vpp/extras/gomemif/memif"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
)

// Transport is an ndn.Transport that communicates via libmemif.
type Transport interface {
	ndn.Transport

	Locator() Locator
}

// New creates a Transport.
func New(loc Locator) (Transport, error) {
	if e := loc.Validate(); e != nil {
		return nil, fmt.Errorf("loc.Validate %w", e)
	}
	loc.applyDefaults()

	sock, e := memif.NewSocket(os.Args[0], loc.SocketName)
	if e != nil {
		return nil, fmt.Errorf("memif.NewSocket %w", e)
	}

	tr := &transport{
		loc:        loc,
		sock:       sock,
		memifError: make(chan error),
		dataroom:   loc.Dataroom,
		rx:         make(chan []byte, loc.RxQueueSize),
		tx:         make(chan []byte, loc.TxQueueSize),
		notifyRx:   make(chan int),
		notifyTx:   make(chan int),
	}

	a := &memif.Arguments{
		ConnectedFunc:    tr.memifConnected,
		DisconnectedFunc: tr.memifDisconnected,
	}
	loc.toArguments(a)
	tr.intf, e = sock.NewInterface(a)
	if e != nil {
		sock.Delete()
		return nil, fmt.Errorf("sock.NewInterface %w", e)
	}

	tr.sock.StartPolling(tr.memifError)
	tr.intf.RequestConnection()
	go tr.rxLoop()
	go tr.txLoop()
	return tr, nil
}

type transport struct {
	loc        Locator
	sock       *memif.Socket
	memifError chan error
	intf       *memif.Interface

	dataroom int
	rx       chan []byte
	tx       chan []byte
	notifyRx chan int
	notifyTx chan int
}

const (
	notifyClosing = iota
	notifyConnected
	notifyDisconnected
)

func (tr *transport) Locator() Locator {
	return tr.loc
}

func (tr *transport) Rx() <-chan []byte {
	return tr.rx
}

func (tr *transport) Tx() chan<- []byte {
	return tr.tx
}

func (tr *transport) memifConnected(intf *memif.Interface) error {
	tr.notifyRx <- notifyConnected
	tr.notifyTx <- notifyConnected
	return nil
}

func (tr *transport) memifDisconnected(intf *memif.Interface) error {
	tr.notifyRx <- notifyDisconnected
	tr.notifyTx <- notifyDisconnected
	return nil
}

func (tr *transport) rxLoop() {
	buf := make([]byte, tr.dataroom)
	var rxq *memif.Queue
	for {
		select {
		case a := <-tr.notifyRx:
			switch a {
			case notifyClosing:
				goto CLOSE
			case notifyConnected:
				rxq, _ = tr.intf.GetRxQueue(0)
			case notifyDisconnected:
				rxq = nil
			}
		default:
		}
		if rxq == nil {
			runtime.Gosched()
			continue
		}

		n, e := rxq.ReadPacket(buf)
		if e == nil && n > 14 {
			select {
			case tr.rx <- buf[14:n]:
				buf = make([]byte, tr.dataroom)
			default:
			}
		}
	}

CLOSE:
	go func() {
		for range tr.notifyRx {
		}
	}()

	close(tr.rx)
	tr.sock.StopPolling()
	tr.sock.Delete()
	close(tr.notifyRx)
	close(tr.notifyTx)
}

func (tr *transport) txLoop() {
	var eth layers.Ethernet
	eth.SrcMAC = AddressApp
	eth.DstMAC = AddressDPDK
	eth.EthernetType = packettransport.EthernetTypeNDN
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}

	var txq *memif.Queue
	for {
		select {
		case a := <-tr.notifyTx:
			switch a {
			case notifyConnected:
				txq, _ = tr.intf.GetTxQueue(0)
			case notifyDisconnected:
				txq = nil
			}
		case <-tr.memifError:
			goto CLOSE
		case pkt, ok := <-tr.tx:
			if !ok {
				goto CLOSE
			}
			if txq != nil {
				if e := gopacket.SerializeLayers(buf, opts, &eth, gopacket.Payload(pkt)); e != nil {
					continue
				}
				txq.WritePacket(buf.Bytes())
			}
		}
	}

CLOSE:
	tr.notifyRx <- notifyClosing
	for range tr.notifyTx {
	}
}
