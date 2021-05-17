package l3

import (
	"encoding/binary"
	"errors"
	"io"

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

func tokenInsertID(oldToken []byte, id uint16) (token []byte) {
	token = make([]byte, 2, 2+len(oldToken))
	binary.LittleEndian.PutUint16(token, id)
	return append(token, oldToken...)
}

func tokenStripID(token []byte) (id uint16, newToken []byte) {
	id = binary.LittleEndian.Uint16(token)
	newToken = token[2:]
	return
}

type fwFace struct {
	Face
	fw            *forwarder
	id            uint16
	routes        map[string]ndn.Name
	announcements map[string]ndn.Name
}

func (f *fwFace) rxLoop() {
	for pkt := range f.Rx() {
		switch {
		case pkt.Interest != nil:
			pkt.Lp.PitToken = tokenInsertID(pkt.Lp.PitToken, f.id)
			f.fw.pkt <- pkt
		case pkt.Data != nil, pkt.Nack != nil:
			f.fw.pkt <- pkt
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

func (f *fwFace) lpmRoute(name ndn.Name) int {
	for _, name := range f.routes {
		if name.IsPrefixOf(name) {
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

		if !f.fw.announcements.ContainsKey(nameS) {
			for dest := range f.fw.readvertise {
				dest.Advertise(name)
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
	if !f.fw.announcements.ContainsKey(nameS) {
		for dest := range f.fw.readvertise {
			dest.Withdraw(name)
		}
	}
}

func (f *fwFace) Close() error {
	f.fw.execute(func() {
		for nameS, name := range f.announcements {
			f.removeAnnouncementImpl(name, nameS)
		}
		delete(f.fw.faces, f.id)
		close(f.Tx())
	})
	return nil
}
