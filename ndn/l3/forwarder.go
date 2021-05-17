package l3

import (
	"math/rand"
	"sync"

	"github.com/jwangsadinata/go-multimap"
	"github.com/jwangsadinata/go-multimap/setmultimap"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// Forwarder is a logical forwarding plane.
// Its main purpose is to demultiplex incoming packets among faces, where a 'face' is defined as a duplex stream of packets.
//
// This is a simplified forwarder with several limitations.
//  - There is no loop prevention: no Nonce list and no decrementing HopLimit.
//    If multiple uplinks have "/" route, Interests will be forwarded among them and might cause persistent loops.
//    Thus, it is not recommended to connect to multiple uplinks.
//  - There is no pending Interest table. Instead, downstream 'face' ID is inserted as part of the PIT token.
//    Since PIT token cannot exceed 32 octets, this takes away some space.
//    Thus, consumers are allowed to use a PIT token up to 30 octets; Interests with longer PIT tokens may be dropped.
type Forwarder interface {
	// AddFace adds a Face to the forwarder.
	// face.Rx() and face.Tx() should not be used after this operation.
	AddFace(face Face) (FwFace, error)

	// AddReadvertiseDestination adds a destination for prefix announcement.
	//
	// Limitations of current implementation:
	//  - Existing announcements are not advertised on dest.
	//    Thus, it is recommended to add all readvertise destinations before announcing a prefix.
	//  - There is no error handling.
	AddReadvertiseDestination(dest ReadvertiseDestination)

	// RemoveReadvertiseDestination removes a destination for prefix announcement.
	//
	// Limitations of current implementation:
	//  - Announcements are not withdrawn before removing dest.
	//  - There is no error handling.
	RemoveReadvertiseDestination(dest ReadvertiseDestination)
}

// NewForwarder creates a Forwarder.
func NewForwarder() Forwarder {
	fw := &forwarder{
		faces:         make(map[uint16]*fwFace),
		announcements: setmultimap.New(),
		readvertise:   make(map[ReadvertiseDestination]bool),
		cmd:           make(chan func()),
		pkt:           make(chan *ndn.Packet),
	}
	go fw.loop()
	return fw
}

type forwarder struct {
	faces         map[uint16]*fwFace
	announcements multimap.MultiMap // multimap[string(prefixV)]*fwFace
	readvertise   map[ReadvertiseDestination]bool
	cmd           chan func()
	pkt           chan *ndn.Packet
}

func (fw *forwarder) AddFace(face Face) (ff FwFace, e error) {
	f := &fwFace{
		Face:          face,
		fw:            fw,
		routes:        make(map[string]ndn.Name),
		announcements: make(map[string]ndn.Name),
	}

	fw.execute(func() {
		if len(fw.faces) >= MaxFwFaces {
			e = ErrMaxFwFaces
			f = nil
			return
		}

		for f.id == 0 || fw.faces[f.id] != nil {
			f.id = uint16(rand.Uint32())
		}
		fw.faces[f.id] = f
	})

	go f.rxLoop()
	return f, e
}

func (fw *forwarder) AddReadvertiseDestination(dest ReadvertiseDestination) {
	fw.execute(func() {
		if fw.readvertise[dest] {
			return
		}
		fw.readvertise[dest] = true
	})
}

func (fw *forwarder) RemoveReadvertiseDestination(dest ReadvertiseDestination) {
	fw.execute(func() {
		if !fw.readvertise[dest] {
			return
		}
		delete(fw.readvertise, dest)
	})
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
	lpmLen := 0
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
	id, token := tokenStripID(pkt.Lp.PitToken)
	if f := fw.faces[id]; f != nil {
		pkt.Lp.PitToken = token
		f.Tx() <- pkt
	}
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
