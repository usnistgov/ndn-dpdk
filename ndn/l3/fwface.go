package l3

import (
	"encoding/binary"
	"errors"
	"io"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/ndn"
)

// MaxFwFaces is the maximum number of active FwFaces in a Forwarder.
const MaxFwFaces = 1 << 23

// Error conditions.
var (
	ErrMaxFwFaces = errors.New("too many FwFaces")
)

// FwFace represents a face added to the forwarder.
type FwFace interface {
	io.Closer
	Transport() Transport
	State() TransportState
	OnStateChange(cb func(st TransportState)) (cancel func())

	AddRoute(name ndn.Name)
	RemoveRoute(name ndn.Name)

	AddAnnouncement(name ndn.Name)
	RemoveAnnouncement(name ndn.Name)
}

const tokenSize = int(unsafe.Sizeof(uint32(0)))

func tokenInsertID(oldToken []byte, id uint32) (token []byte) {
	token = make([]byte, tokenSize, tokenSize+len(oldToken))
	binary.LittleEndian.PutUint32(token, id)
	return append(token, oldToken...)
}

func tokenStripID(token []byte) (id uint32, newToken []byte) {
	if len(token) < tokenSize {
		return 0, nil
	}
	id = binary.LittleEndian.Uint32(token)
	newToken = token[tokenSize:]
	return
}

type fwFace struct {
	Face
	fw            *forwarder
	id            uint32
	tx            chan<- ndn.L3Packet
	routes        map[string]ndn.Name
	announcements map[string]ndn.Name
}

func (f *fwFace) rxLoop() {
	for pkt := range f.Rx() {
		switch {
		case pkt.Interest != nil:
			pkt.Lp.PitToken = tokenInsertID(pkt.Lp.PitToken, f.id)
		case pkt.Data != nil, pkt.Nack != nil:
		default:
			continue
		}
		f.fw.rx <- fwRxPkt{
			Packet: pkt,
			rxFace: f,
		}
	}
}

func (f *fwFace) AddRoute(name ndn.Name) {
	nameV, _ := name.MarshalBinary()
	nameS := string(nameV)
	f.fw.execute(func() {
		f.routes[nameS] = name
	})
}

func (f *fwFace) RemoveRoute(name ndn.Name) {
	nameV, _ := name.MarshalBinary()
	nameS := string(nameV)
	f.fw.execute(func() {
		delete(f.routes, nameS)
	})
}

func (f *fwFace) lpmRoute(query ndn.Name) int {
	for _, name := range f.routes {
		if name.IsPrefixOf(query) {
			return len(name)
		}
	}
	return -1
}

func (f *fwFace) AddAnnouncement(name ndn.Name) {
	nameV, _ := name.MarshalBinary()
	nameS := string(nameV)
	f.fw.execute(func() {
		f.announcements[nameS] = name

		if !f.fw.announcements.Has(nameS) {
			for dest := range f.fw.readvertise {
				go dest.Advertise(name)
			}
		}
		f.fw.announcements.Put(nameS, f)
	})
}

func (f *fwFace) RemoveAnnouncement(name ndn.Name) {
	nameV, _ := name.MarshalBinary()
	nameS := string(nameV)
	f.fw.execute(func() {
		f.removeAnnouncementImpl(name, nameS)
	})
}

func (f *fwFace) removeAnnouncementImpl(name ndn.Name, nameS string) {
	delete(f.announcements, nameS)

	f.fw.announcements.Remove(nameS, f)
	if !f.fw.announcements.Has(nameS) {
		for dest := range f.fw.readvertise {
			go dest.Withdraw(name)
		}
	}
}

func (f *fwFace) Close() error {
	f.fw.execute(func() {
		for nameS, name := range f.announcements {
			f.removeAnnouncementImpl(name, nameS)
		}
		delete(f.fw.faces, f.id)
		close(f.tx)
	})
	return nil
}
