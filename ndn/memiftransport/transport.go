//go:build linux

// Package memiftransport implements a transport over a shared memory packet interface (memif).
package memiftransport

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

// Transport is an l3.Transport that communicates via libmemif.
type Transport interface {
	l3.Transport

	Locator() Locator
}

// New creates a Transport.
func New(loc Locator) (t Transport, e error) {
	if e := loc.Validate(); e != nil {
		return nil, fmt.Errorf("loc.Validate %w", e)
	}
	loc.ApplyDefaults(RoleClient)

	tr := &transport{}
	tr.TransportBase, tr.p = l3.NewTransportBase(l3.TransportBaseConfig{
		MTU: loc.Dataroom,
	})

	if tr.handle, e = newHandle(loc, tr.p.SetState); e != nil {
		return nil, e
	}
	return tr, nil
}

type transport struct {
	*handle
	*l3.TransportBase
	p *l3.TransportBasePriv
}

func (tr *transport) Locator() Locator {
	return tr.loc
}
