// Package l3 defines a network layer face abstraction.
//
// The Transport interface defines a lower layer communication channel.
// It knows NDN-TLV structure, but not NDN packet types.
// It should be implemented for different communication technologies.
// NDNgo library offers Transport implementations for memif, UDP sockets, etc.
//
// The Face type is the service exposed to the network layer.
// It allows sending and receiving packets on a Transport.
package l3

import (
	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Limits and defaults.
const (
	MinReassemblerCapacity = 16
)

// FaceConfig contains options for NewFace.
type FaceConfig struct {
	ReassemblerCapacity int
}

func (cfg *FaceConfig) applyDefaults() {
	cfg.ReassemblerCapacity = math.MaxInt(cfg.ReassemblerCapacity, MinReassemblerCapacity)
}

// Face represents a communicate channel to send and receive NDN network layer packets.
type Face interface {
	// Transport returns the underlying transport.
	Transport() Transport

	// Rx returns a channel to receive incoming packets toward the forwarder.
	// This function always returns the same channel.
	// This channel is closed when the face is closed.
	Rx() <-chan *ndn.Packet

	// Tx returns a channel to send outgoing packets from the forwarder.
	// This function always returns the same channel.
	// Closing this channel causes the face to close.
	Tx() chan<- ndn.L3Packet

	State() TransportState
	OnStateChange(cb func(st TransportState)) (cancel func())
}

// NewFace creates a Face.
// tr.Rx() and tr.Tx() should not be used after this operation.
func NewFace(tr Transport, cfg FaceConfig) (Face, error) {
	cfg.applyDefaults()
	f := &face{
		faceTr:      faceTr{tr},
		rx:          make(chan *ndn.Packet),
		tx:          make(chan ndn.L3Packet),
		reassembler: ndn.NewLpReassembler(cfg.ReassemblerCapacity),
	}

	if mtu := tr.MTU(); mtu > 0 {
		f.fragmenter = ndn.NewLpFragmenter(mtu)
	}

	go f.rxLoop()
	go f.txLoop()
	return f, nil
}

type face struct {
	faceTr
	rx chan *ndn.Packet
	tx chan ndn.L3Packet

	fragmenter  *ndn.LpFragmenter
	reassembler *ndn.LpReassembler
}

type faceTr struct {
	Transport
}

func (f *face) Transport() Transport {
	return f.faceTr.Transport
}

func (f *face) Rx() <-chan *ndn.Packet {
	return f.rx
}

func (f *face) Tx() chan<- ndn.L3Packet {
	return f.tx
}

func (f *face) rxLoop() {
	for wire := range f.faceTr.Rx() {
		var pkt ndn.Packet
		e := tlv.Decode(wire, &pkt)
		if e != nil {
			continue
		}

		if pkt.Fragment == nil {
			f.rx <- &pkt
		} else {
			full, e := f.reassembler.Accept(&pkt)
			if e == nil && full != nil {
				f.rx <- full
			}
		}
	}
	close(f.rx)
}

func (f *face) txLoop() {
	transportTx := f.faceTr.Tx()
	for l3packet := range f.tx {
		pkt := l3packet.ToPacket()

		if f.fragmenter == nil {
			f.txFrames(transportTx, pkt)
		} else {
			frags, e := f.fragmenter.Fragment(pkt)
			if e == nil {
				f.txFrames(transportTx, frags...)
			}
		}
	}
	close(transportTx)
}

func (f *face) txFrames(transportTx chan<- []byte, frames ...*ndn.Packet) {
	for _, frame := range frames {
		wire, e := tlv.EncodeFrom(frame)
		if e == nil {
			transportTx <- wire
		}
	}
}
