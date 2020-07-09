package intface

import (
	"net"

	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/socketface"
	"github.com/usnistgov/ndn-dpdk/ndn"
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
	A ndn.L3Face

	// Rx is application side RX channel.
	// It's equivalent to A.Rx().
	Rx <-chan *ndn.Packet

	// Tx is application side TX channel.
	// It's equivalent to A.Tx().
	Tx chan<- ndn.L3Packet
}

// New creates an IntFace.
func New() (*IntFace, error) {
	var f IntFace

	connA, connD := net.Pipe()
	trA, e := sockettransport.New(connA, sockettransport.Config{})
	if e != nil {
		return nil, e
	}
	trD, e := sockettransport.New(connD, sockettransport.Config{})
	if e != nil {
		return nil, e
	}

	if f.A, e = ndn.NewL3Face(trA); e != nil {
		return nil, e
	}
	if f.D, e = socketface.Wrap(trD, socketface.Config{}); e != nil {
		return nil, e
	}

	f.ID = f.D.ID()
	f.Rx = f.A.Rx()
	f.Tx = f.A.Tx()
	return &f, nil
}

// MustNew creates an IntFace, and panics on error.
func MustNew() *IntFace {
	f, e := New()
	if e != nil {
		panic(e)
	}
	return f
}

// SetDown changes up/down state on the DPDK side.
func (f *IntFace) SetDown(isDown bool) {
	f.D.SetDown(isDown)
}
