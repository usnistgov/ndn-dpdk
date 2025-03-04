package upf

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"

	"github.com/usnistgov/ndn-dpdk/core/uintalloc"
	"github.com/wmnsk/go-pfcp/ie"
	"github.com/wmnsk/go-pfcp/message"
)

// Session represents a PFCP session and the associated face.
type Session struct {
	CpSEID, UpSEID uint64
	Parser         SessionParser
	FaceID         string

	sct SessionTeids
}

// SessionTable stores PFCP sessions and instructs face creation.
type SessionTable struct {
	n3ipv4      net.IP              // UPF N3 IPv4
	table       map[uint64]*Session // UpSEID => Session
	teidChooser *TeidChooser
	createFace  func(ctx context.Context, loc SessionLocatorFields) (id string, e error)
	destroyFace func(ctx context.Context, id string) error
}

// EstablishmentRequest handles a SessionEstablishmentRequest message.
func (st *SessionTable) EstablishmentRequest(ctx context.Context, req *message.SessionEstablishmentRequest, rspIEs []*ie.IE) (sess *Session, rspIEsRet []*ie.IE, e error) {
	cpfSEID, e := req.CPFSEID.FSEID()
	if e != nil {
		return nil, rspIEs, fmt.Errorf("CP F-SEID: %w", e)
	}

	sess = &Session{
		CpSEID: cpfSEID.SEID,
		UpSEID: uintalloc.Alloc64(st.table),
		sct:    SessionTeids{},
	}
	sess.Parser.ChooseTeid = func(fTEID *ie.FTEIDFields) *ie.FTEIDFields {
		teid := st.teidChooser.Alloc(fTEID, sess.sct)
		rsp := ie.NewFTEIDFields(0, teid, st.n3ipv4, nil, 0)
		rsp.SetIPv4Flag()
		return rsp
	}
	st.table[sess.UpSEID] = sess

	if rspIEs, e = sess.Parser.EstablishmentRequest(req, rspIEs); e != nil {
		return sess, rspIEs, e
	}
	return sess, rspIEs, st.createFaceWhenReady(ctx, sess)
}

// ModificationRequest handles a SessionModificationRequest message.
func (st *SessionTable) ModificationRequest(ctx context.Context, req *message.SessionModificationRequest, rspIEs []*ie.IE) (sess *Session, rspIEsRet []*ie.IE, e error) {
	if sess = st.table[req.SEID()]; sess == nil {
		return nil, rspIEs, os.ErrNotExist
	}

	if rspIEs, e = sess.Parser.ModificationRequest(req, rspIEs); e != nil {
		return sess, rspIEs, e
	}
	return sess, rspIEs, st.createFaceWhenReady(ctx, sess)
}

func (st *SessionTable) createFaceWhenReady(ctx context.Context, sess *Session) error {
	loc, ok := sess.Parser.LocatorFields()
	if !ok {
		return nil
	}

	id, e := st.createFace(ctx, loc)
	if e != nil {
		return e
	}
	sess.FaceID = id
	return nil
}

// DeletionRequest handles a SessionDeletionRequest message.
func (st *SessionTable) DeletionRequest(ctx context.Context, req *message.SessionDeletionRequest) (sess *Session, e error) {
	if sess = st.table[req.SEID()]; sess == nil {
		return nil, os.ErrNotExist
	}
	defer func() {
		st.teidChooser.Free(sess.sct)
		delete(st.table, sess.UpSEID)
	}()

	if sess.FaceID != "" {
		e = st.destroyFace(ctx, sess.FaceID)
	}
	return
}

// NewSessionTable constructs SessionTable.
func NewSessionTable(
	n3ipv4 netip.Addr,
	createFace func(ctx context.Context, sloc SessionLocatorFields) (id string, e error),
	destroyFace func(ctx context.Context, id string) error,
) *SessionTable {
	n3ipv4b := n3ipv4.As4()
	return &SessionTable{
		n3ipv4:      n3ipv4b[:],
		table:       map[uint64]*Session{},
		teidChooser: NewTeidChooser(),
		createFace:  createFace,
		destroyFace: destroyFace,
	}
}
