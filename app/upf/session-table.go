package upf

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"

	"github.com/wmnsk/go-pfcp/message"
)

// Session represents a PFCP session and the associated face.
type Session struct {
	CpSEID, UpSEID uint64
	Parser         SessionParser
	FaceID         string
}

// FaceCreator knows how to create and destroy GTP-U face.
type FaceCreator interface {
	CreateFace(ctx context.Context, loc SessionLocatorFields) (id string, e error)
	DestroyFace(ctx context.Context, id string) error
}

// SessionTable stores PFCP sessions and instructs face creation.
type SessionTable struct {
	table map[uint64]*Session
	fc    FaceCreator
}

// EstablishmentRequest handles a SessionEstablishmentRequest message.
func (st *SessionTable) EstablishmentRequest(ctx context.Context, req *message.SessionEstablishmentRequest) (sess *Session, e error) {
	cpfSEID, e := req.CPFSEID.FSEID()
	if e != nil {
		return nil, fmt.Errorf("CP F-SEID: %w", e)
	}

	sess = &Session{
		CpSEID: cpfSEID.SEID,
	}
	for sess.UpSEID == 0 || st.table[sess.UpSEID] != nil {
		sess.UpSEID = rand.Uint64()
	}
	st.table[sess.UpSEID] = sess

	if e := sess.Parser.EstablishmentRequest(req); e != nil {
		return sess, e
	}
	return sess, st.createFaceWhenReady(ctx, sess)
}

// ModificationRequest handles a SessionModificationRequest message.
func (st *SessionTable) ModificationRequest(ctx context.Context, req *message.SessionModificationRequest) (sess *Session, e error) {
	if sess = st.table[req.SEID()]; sess == nil {
		return nil, os.ErrNotExist
	}

	if e := sess.Parser.ModificationRequest(req); e != nil {
		return sess, e
	}
	return sess, st.createFaceWhenReady(ctx, sess)
}

func (st *SessionTable) createFaceWhenReady(ctx context.Context, sess *Session) error {
	loc, ok := sess.Parser.LocatorFields()
	if !ok {
		return nil
	}

	id, e := st.fc.CreateFace(ctx, loc)
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

	defer delete(st.table, sess.UpSEID)
	if sess.FaceID == "" {
		return sess, nil
	}
	return sess, st.fc.DestroyFace(ctx, sess.FaceID)
}

// NewSessionTable constructs SessionTable.
func NewSessionTable(fc FaceCreator) *SessionTable {
	return &SessionTable{
		table: map[uint64]*Session{},
		fc:    fc,
	}
}
