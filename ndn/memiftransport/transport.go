//go:build linux

// Package memiftransport implements a transport over a shared memory packet interface (memif).
package memiftransport

import (
	"fmt"
	"io"

	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

// Transport is an l3.Transport that communicates via libmemif.
type Transport interface {
	l3.Transport

	Locator() Locator
}

// New creates a Transport.
func New(loc Locator) (Transport, error) {
	if e := loc.Validate(); e != nil {
		return nil, fmt.Errorf("loc.Validate %w", e)
	}
	loc.ApplyDefaults(RoleClient)

	tr := &transport{}
	tr.TransportBase, tr.p = l3.NewTransportBase(l3.TransportBaseConfig{
		TransportQueueConfig: loc.TransportQueueConfig,
		MTU:                  loc.Dataroom,
	})

	hdl, e := newHandle(loc, tr.p.SetState)
	if e != nil {
		return nil, e
	}
	tr.hdl = hdl

	go tr.rxLoop()
	go tr.txLoop()
	return tr, nil
}

type transport struct {
	*l3.TransportBase
	p   *l3.TransportBasePriv
	hdl *handle
}

func (tr *transport) Locator() Locator {
	return tr.hdl.Locator
}

func (tr *transport) rxLoop() {
	dataroom := tr.MTU()
	buf := make([]byte, dataroom)
	for {
		n, e := tr.hdl.Read(buf)
		if e == io.EOF {
			break
		}
		if e != nil {
			continue
		}

		select {
		case tr.p.Rx <- buf[:n]:
		default: // drop
		}
		buf = make([]byte, dataroom)
	}
	close(tr.p.Rx)
}

func (tr *transport) txLoop() {
	for pkt := range tr.p.Tx {
		tr.hdl.Write(pkt)
	}
	tr.hdl.Close()
}
