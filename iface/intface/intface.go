// Package intface implements an internal face for internal applications.
// It bridges a iface.Face (socket face) on DPDK side and an ndn.L3Face on application side.
package intface

import (
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/socketface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
)

// IntFace is an iface.Face and a ndn.L3Face connected together.
type IntFace struct {
	// D is the face on DPDK side.
	// Packets sent on D are received on A.
	D iface.Face

	// ID is the ID on DPDK side.
	ID iface.ID

	// A is the face on application side.
	// Packets sent on A are received by D.
	A l3.Face

	// Rx is application side RX channel.
	// It's equivalent to A.Rx().
	Rx <-chan *ndn.Packet

	// Tx is application side TX channel.
	// It's equivalent to A.Tx().
	Tx chan<- ndn.L3Packet
}

// New creates an IntFace.
func New(cfg socketface.Config) (*IntFace, error) {
	var f IntFace

	trA, trD, e := sockettransport.Pipe(sockettransport.Config{})
	if e != nil {
		return nil, e
	}

	if f.A, e = l3.NewFace(trA); e != nil {
		return nil, e
	}
	if f.D, e = socketface.Wrap(trD, cfg); e != nil {
		return nil, e
	}

	f.ID = f.D.ID()
	f.Rx = f.A.Rx()
	f.Tx = f.A.Tx()
	return &f, nil
}

// Must panics on error.
func Must(f *IntFace, e error) *IntFace {
	if e != nil {
		panic(e)
	}
	return f
}

// MustNew creates an IntFace with default settings, and panics on error.
func MustNew() *IntFace {
	return Must(New(socketface.Config{}))
}

// SetDown changes up/down state on the DPDK side.
func (f *IntFace) SetDown(isDown bool) {
	f.D.SetDown(isDown)
}
