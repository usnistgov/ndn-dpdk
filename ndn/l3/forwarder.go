package l3

import (
	"math/rand"
	"sync"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/zyedidia/generic/multimap"
)

// Forwarder is a logical forwarding plane.
// Its main purpose is to demultiplex incoming packets among faces, where a 'face' is defined as a duplex stream of packets.
//
// This is a simplified forwarder with several limitations.
//   - There is no loop prevention: no Nonce list and no decrementing HopLimit.
//     If multiple uplinks have "/" route, Interests will be forwarded among them and might cause persistent loops.
//     Thus, it is not recommended to connect to multiple uplinks with overlapping routes.
//   - There is no pending Interest table. Instead, downstream 'face' ID is inserted as part of the PIT token.
//     Since PIT token cannot exceed 32 octets, this takes away some space.
//     Thus, consumers are allowed to use a PIT token up to 28 octets; Interests with longer PIT tokens may be dropped.
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
		faces:         map[uint32]*fwFace{},
		announcements: multimap.NewMapSlice[string, *fwFace](),
		readvertise:   map[ReadvertiseDestination]bool{}, // cannot use mapset because ReadvertiseDestination is not 'comparable'
		cmd:           make(chan func()),
		rx:            make(chan fwRxPkt),
	}
	go fw.loop()
	return fw
}

type fwRxPkt struct {
	*ndn.Packet
	rxFace *fwFace
}

type forwarder struct {
	faces         map[uint32]*fwFace
	announcements multimap.MultiMap[string, *fwFace]
	readvertise   map[ReadvertiseDestination]bool
	cmd           chan func()
	rx            chan fwRxPkt
}

func (fw *forwarder) AddFace(face Face) (ff FwFace, e error) {
	f := &fwFace{
		Face:          face,
		fw:            fw,
		tx:            face.Tx(),
		routes:        map[string]ndn.Name{},
		announcements: map[string]ndn.Name{},
	}

	fw.do(func() {
		if len(fw.faces) >= MaxFwFaces {
			e = ErrMaxFwFaces
			f = nil
			return
		}

		for f.id == 0 || fw.faces[f.id] != nil {
			f.id = rand.Uint32()
		}
		fw.faces[f.id] = f
	})

	if e != nil {
		return nil, e
	}
	go f.rxLoop()
	return f, nil
}

func (fw *forwarder) AddReadvertiseDestination(dest ReadvertiseDestination) {
	fw.do(func() {
		if fw.readvertise[dest] {
			return
		}
		fw.readvertise[dest] = true
	})
}

func (fw *forwarder) RemoveReadvertiseDestination(dest ReadvertiseDestination) {
	fw.do(func() {
		if !fw.readvertise[dest] {
			return
		}
		delete(fw.readvertise, dest)
	})
}

func (fw *forwarder) do(fn func()) {
	done := make(chan struct{})
	fw.cmd <- func() {
		defer close(done)
		fn()
	}
	<-done
}

func (fw *forwarder) loop() {
	for {
		select {
		case fn := <-fw.cmd:
			fn()
		case pkt := <-fw.rx:
			switch {
			case pkt.Interest != nil:
				fw.forwardInterest(pkt)
			case pkt.Data != nil, pkt.Nack != nil:
				fw.forwardDataNack(pkt)
			}
		}
	}
}

func (fw *forwarder) forwardInterest(pkt fwRxPkt) {
	lpmLen := 0
	var nexthops []*fwFace
	for _, f := range fw.faces {
		if pkt.rxFace == f {
			continue
		}

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
		f.tx <- pkt
	}
}

func (fw *forwarder) forwardDataNack(pkt fwRxPkt) {
	var id uint32
	id, pkt.Lp.PitToken = tokenStripID(pkt.Lp.PitToken)
	if f := fw.faces[id]; f != nil {
		f.tx <- pkt.Packet
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
