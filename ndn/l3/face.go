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
	"fmt"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Limits and defaults.
const (
	MinReassemblerCapacity = 16
	DefaultRxQueueSize     = 64
	DefaultTxQueueSize     = 64
)

// FaceConfig contains options for NewFace.
type FaceConfig struct {
	// ReassemblerCapacity is the maximum number of partial messages stored in the reassembler.
	// Default is MinReassemblerCapacity.
	ReassemblerCapacity int

	// RxQueueSize is the Go channel buffer size of RX channel.
	// Default is DefaultRxQueueSize.
	RxQueueSize int `json:"rxQueueSize,omitempty"`

	// TxQueueSize is the Go channel buffer size of TX channel.
	// Default is DefaultTxQueueSize.
	TxQueueSize int `json:"txQueueSize,omitempty"`
}

func (cfg *FaceConfig) applyDefaults() {
	cfg.ReassemblerCapacity = max(cfg.ReassemblerCapacity, MinReassemblerCapacity)
	if cfg.RxQueueSize <= 0 {
		cfg.RxQueueSize = DefaultRxQueueSize
	}
	if cfg.TxQueueSize <= 0 {
		cfg.TxQueueSize = DefaultTxQueueSize
	}
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
// tr.Read() and tr.Write() should not be used after this operation.
func NewFace(tr Transport, cfg FaceConfig) (Face, error) {
	cfg.applyDefaults()
	mtu := tr.MTU()
	if mtu <= 0 {
		return nil, fmt.Errorf("bad MTU %d", mtu)
	}

	f := &face{
		faceTr:      faceTr{tr},
		rx:          make(chan *ndn.Packet, cfg.RxQueueSize),
		tx:          make(chan ndn.L3Packet, cfg.TxQueueSize),
		mtu:         mtu,
		fragmenter:  ndn.NewLpFragmenter(mtu),
		reassembler: ndn.NewLpReassembler(cfg.ReassemblerCapacity),
	}
	go f.rxLoop()
	go f.txLoop()
	return f, nil
}

type face struct {
	faceTr
	rx chan *ndn.Packet
	tx chan ndn.L3Packet

	mtu         int
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
	buf := make([]byte, f.mtu)
	for {
		n, e := f.faceTr.Read(buf)
		if e != nil {
			break
		}
		if n == 0 {
			continue
		}

		var pkt ndn.Packet
		if e := tlv.Decode(buf[:n], &pkt); e != nil {
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

		buf = make([]byte, f.mtu)
	}
	close(f.rx)
}

func (f *face) txLoop() {
	for l3packet := range f.tx {
		pkt := l3packet.ToPacket()
		frames, e := f.fragmenter.Fragment(pkt)
		if e != nil {
			continue
		}

		for _, frame := range frames {
			if wire, e := tlv.EncodeFrom(frame); e == nil {
				f.faceTr.Write(wire)
			}
		}
	}
	f.faceTr.Close()
}
