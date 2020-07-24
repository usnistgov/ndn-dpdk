// Package memiftransport implements a transport over a shared memory packet interface (memif).
package memiftransport

import (
	"fmt"

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

	hdl, e := newHandle(loc)
	if e != nil {
		return nil, e
	}

	pcfg := packettransport.Config{
		Locator: packettransport.Locator{
			Local:  AddressApp,
			Remote: AddressDPDK,
		},
		TransportQueueConfig: loc.TransportQueueConfig,
	}
	ptr, e := packettransport.New(hdl, pcfg)

	return &transport{
		Transport: ptr,
		loc:       loc,
	}, nil
}

type transport struct {
	packettransport.Transport
	loc Locator
}

func (tr *transport) Locator() Locator {
	return tr.loc
}
