package l3

import (
	"errors"
	"io"
	"math/rand"
	"sync"

	"github.com/usnistgov/ndn-dpdk/ndn"
)

const (
	tokenBits       = 64
	tokenSuffixBits = 48
	tokenPrefixBits = tokenBits - tokenSuffixBits
	tokenSuffixMask = 1<<tokenSuffixBits - 1
	tokenPrefixMask = (1<<tokenBits - 1) - tokenSuffixMask

	// MaxFwFaces is the maximum number of active FwFaces in a Forwarder.
	MaxFwFaces = 1 << tokenPrefixBits / 4
)

// Error conditions.
var (
	ErrMaxFwFaces = errors.New("too many FwFaces")
)

var defaultFw Forwarder

// FwFace represents a face added to the forwarder.
type FwFace interface {
	io.Closer
	Transport() Transport
	State() TransportState
	OnStateChange(cb func(st TransportState)) io.Closer

	AddRoute(prefix ndn.Name)
	RemoveRoute(prefix ndn.Name)
}

// Forwarder is a logical forwarding plane.
// Its main purpose is to demultiplex incoming packets among faces, where a 'face' is defined as a duplex stream of packets.
//
// This is a simplified forwarder with several limitations.
// There is no loop prevention, so it is not recommended to connect multiple uplinks with "/" route simultaneously.
// Nack handling is incomplete: if any nexthop replies a Nack, it is delivered to the consumer without waiting for other nexthops.
type Forwarder interface {
	// AddTransport constructs a Face and invokes AddFace.
	AddTransport(tr Transport) (FwFace, error)

	// AddFace adds a Face to the forwarder.
	// face.Rx() and face.Tx() should not be used after this operation.
	AddFace(face Face) (FwFace, error)
}

// NewForwarder creates a Forwarder.
func NewForwarder() Forwarder {
	fw := &forwarder{
		faces: make(map[uint64]*fwFace),
		cmd:   make(chan func()),
		pkt:   make(chan *ndn.Packet),
	}
	go fw.loop()
	return fw
}

type forwarder struct {
	faces map[uint64]*fwFace
	cmd   chan func()
	pkt   chan *ndn.Packet
}

func (fw *forwarder) AddTransport(tr Transport) (FwFace, error) {
	face, e := NewFace(tr)
	if e != nil {
		return nil, e
	}
	return fw.AddFace(face)
}

func (fw *forwarder) AddFace(face Face) (ff FwFace, e error) {
	f := &fwFace{
		Face:        face,
		fw:          fw,
		tokenSuffix: rand.Uint64(),
		routes:      make(map[string]ndn.Name),
	}

	fw.execute(func() {
		if len(fw.faces) >= MaxFwFaces {
			e = ErrMaxFwFaces
			f = nil
			return
		}

		for f.tokenPrefix == 0 || fw.faces[f.tokenPrefix] != nil {
			f.tokenPrefix = rand.Uint64() << tokenSuffixBits
		}
		fw.faces[f.tokenPrefix] = f
	})

	go f.rxLoop()
	return f, e
}

func (fw *forwarder) execute(fn func()) {
	done := make(chan bool)
	fw.cmd <- func() {
		fn()
		done <- true
	}
	<-done
}

func (fw *forwarder) loop() {
	for {
		select {
		case fn := <-fw.cmd:
			fn()
		case pkt := <-fw.pkt:
			switch {
			case pkt.Interest != nil:
				fw.forwardInterest(pkt)
			case pkt.Data != nil, pkt.Nack != nil:
				fw.forwardDataNack(pkt)
			}
		}
	}
}

func (fw *forwarder) forwardInterest(pkt *ndn.Packet) {
	lpmLen := -1
	var nexthops []*fwFace
	for _, f := range fw.faces {
		matchLen := f.lpmRoute(pkt.Interest.Name)
		switch {
		case matchLen > lpmLen:
			lpmLen = matchLen
			nexthops = nil
			fallthrough
		case matchLen == lpmLen:
			nexthops = append(nexthops, f)
		}
	}

	for _, f := range nexthops {
		f.Tx() <- pkt
	}
}

func (fw *forwarder) forwardDataNack(pkt *ndn.Packet) {
	token := ndn.PitTokenToUint(pkt.Lp.PitToken)
	tokenPrefix := token & tokenPrefixMask
	if f := fw.faces[tokenPrefix]; f != nil {
		f.Tx() <- pkt
	}
}

type fwFace struct {
	Face
	fw          *forwarder
	tokenPrefix uint64
	tokenSuffix uint64
	routes      map[string]ndn.Name
}

func (f *fwFace) rxLoop() {
	for pkt := range f.Rx() {
		switch {
		case pkt.Interest != nil:
			f.tokenSuffix++
			pkt.Lp.PitToken = ndn.PitTokenFromUint(f.tokenPrefix | (f.tokenSuffix & tokenSuffixMask))
			f.fw.pkt <- pkt
		case pkt.Data != nil, pkt.Nack != nil:
			f.fw.pkt <- pkt
		}
	}
}

func (f *fwFace) AddRoute(prefix ndn.Name) {
	prefixV, _ := prefix.MarshalBinary()
	f.fw.execute(func() {
		f.routes[string(prefixV)] = prefix
	})
}

func (f *fwFace) RemoveRoute(prefix ndn.Name) {
	prefixV, _ := prefix.MarshalBinary()
	f.fw.execute(func() {
		delete(f.routes, string(prefixV))
	})
}

func (f *fwFace) lpmRoute(name ndn.Name) int {
	for _, prefix := range f.routes {
		if prefix.IsPrefixOf(name) {
			return len(prefix)
		}
	}
	return -1
}

func (f *fwFace) Close() error {
	f.fw.execute(func() {
		delete(f.fw.faces, f.tokenPrefix)
		close(f.Tx())
	})
	return nil
}

var (
	defaultForwarder     Forwarder
	defaultForwarderOnce sync.Once
)

// GetDefaultForwarder returns the default Forwarder.
func GetDefaultForwarder() Forwarder {
	defaultForwarderOnce.Do(func() {
		defaultForwarder = NewForwarder()
	})
	return defaultForwarder
}

// DeleteDefaultForwarder deletes the default Forwarder.
// This is non-thread-safe and should only be used in test cases.
func DeleteDefaultForwarder() {
	defaultForwarder = nil
	defaultForwarderOnce = sync.Once{}
}

// AddUplink adds a transport to the default Forwarder and sets the route "/" on the face.
func AddUplink(tr Transport) (f FwFace, e error) {
	f, e = GetDefaultForwarder().AddTransport(tr)
	if e != nil {
		f.AddRoute(ndn.Name{})
	}
	return f, e
}
