package main

import (
	"context"
	"os"

	"github.com/usnistgov/ndn-dpdk/app/upf"
	"github.com/wmnsk/go-pfcp/ie"
	"github.com/wmnsk/go-pfcp/message"
	"go.uber.org/zap"
)

type FaceCreator struct{}

var (
	_ upf.FaceCreator = FaceCreator{}
)

// CreateFace implements upf.FaceCreator.
func (FaceCreator) CreateFace(ctx context.Context, sloc upf.SessionLocatorFields) (id string, e error) {
	loc, e := upfCfg.MakeLocator(sloc)
	if e != nil {
		return "", e
	}

	var reply struct {
		ID string `json:"id"`
	}
	e = client.Do(ctx, `
		mutation createFace($locator: JSON!) {
			createFace(locator: $locator) {
				id
			}
		}
	`, map[string]any{
		"locator": loc,
	}, "createFace", &reply)
	if e != nil {
		return "", e
	}
	logger.Info("face created", zap.Any("locator", loc), zap.String("face-id", reply.ID))
	return reply.ID, nil
}

// DestroyFace implements upf.FaceCreator.
func (FaceCreator) DestroyFace(ctx context.Context, id string) error {
	deleted, e := client.Delete(ctx, id)
	if e != nil {
		return e
	}
	logger.Info("face deleted", zap.Bool("deleted", deleted), zap.String("face-id", id))
	return nil
}

var sessTable = upf.NewSessionTable(FaceCreator{})

func handleSessEstab(ctx context.Context, req *message.SessionEstablishmentRequest) (rsp message.Message, e error) {
	sess, e := sessTable.EstablishmentRequest(ctx, req)
	if sess == nil {
		return nil, e
	}
	return message.NewSessionEstablishmentResponse(
		0, 0, sess.CpSEID, req.SequenceNumber, 0,
		upfCfg.UpfNodeID,
		ie.NewCause(ie.CauseRequestAccepted),
		ie.NewFSEID(sess.UpSEID, upfCfg.UpfN4.AsSlice(), nil),
	), e
}

func handleSessMod(ctx context.Context, req *message.SessionModificationRequest) (rsp message.Message, e error) {
	sess, e := sessTable.ModificationRequest(ctx, req)
	if sess == nil {
		return nil, e
	}

	cause := ie.CauseRequestAccepted
	if os.IsNotExist(e) {
		cause = ie.CauseSessionContextNotFound
	}

	return message.NewSessionModificationResponse(
		0, 0, sess.CpSEID, req.SequenceNumber, 0,
		ie.NewCause(cause),
	), nil
}

func handleSessDel(ctx context.Context, req *message.SessionDeletionRequest) (rsp message.Message, e error) {
	sess, e := sessTable.DeletionRequest(ctx, req)
	if sess == nil {
		return nil, e
	}

	cause := ie.CauseRequestAccepted
	if os.IsNotExist(e) {
		cause = ie.CauseSessionContextNotFound
	}

	return message.NewSessionDeletionResponse(
		0, 0, sess.CpSEID, req.SequenceNumber, 0,
		ie.NewCause(cause),
	), nil
}
