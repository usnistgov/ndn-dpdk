package intface

import (
	"net"

	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/socketface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
)

// IntFace is an iface.IFace and a ndn.L3Face connected together.
type IntFace struct {
	// DFace is the face on DPDK side.
	// Packets sent on DFace are received on AFace.
	DFace iface.IFace

	// AFace is the face on application side.
	// Packets sent on AFace are received by DFace.
	AFace ndn.L3Face

	sfD *socketface.SocketFace
	trA *sockettransport.Transport
}

// New creates an IntFace.
func New() (*IntFace, error) {
	var f IntFace

	connD, connA := net.Pipe()
	trD, e := sockettransport.New(connD, sockettransport.Config{})
	if e != nil {
		return nil, e
	}
	f.trA, e = sockettransport.New(connA, sockettransport.Config{})
	if e != nil {
		return nil, e
	}

	f.AFace, e = ndn.NewL3Face(f.trA)
	if e != nil {
		return nil, e
	}

	f.sfD, e = socketface.Wrap(trD, socketface.Config{})
	if e != nil {
		return nil, e
	}
	f.DFace = f.sfD

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
