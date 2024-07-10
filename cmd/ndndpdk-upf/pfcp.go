package main

import (
	"context"
	"errors"
	"reflect"

	"github.com/wmnsk/go-pfcp/ie"
	"github.com/wmnsk/go-pfcp/message"
	"go.uber.org/zap"
)

func pfcpLoop(ctx context.Context) error {
	buf := make([]byte, 9000)
	for {
		n, raddr, e := pfcpConn.ReadFromUDPAddrPort(buf)
		if e != nil {
			return e
		}

		logEntry := logger.With(zap.Stringer("smf", raddr))
		if raddr.Addr() != upfCfg.SmfN4 {
			logEntry.Info("message not from SMF N4 address")
			continue
		}

		wire := buf[:n]
		req, e := message.Parse(wire)
		if e != nil {
			logEntry.Warn("cannot parse message", zap.Error(e), zap.Binary("wire", wire))
			continue
		}
		logEntry = logEntry.With(zap.Stringer("req-type", reflect.TypeOf(req)))

		rsp, e := pfcpDispatch(ctx, logEntry, req)
		if e != nil {
			logEntry.Warn("cannot handle message", zap.Error(e))
		} else if rsp != nil {
			logEntry = logEntry.With(zap.Stringer("rsp-type", reflect.TypeOf(rsp)))
			wire := make([]byte, rsp.MarshalLen())
			if e := rsp.MarshalTo(wire); e != nil {
				logEntry.Warn("cannot encode response", zap.Error(e))
			} else if _, e = pfcpConn.WriteToUDPAddrPort(wire, raddr); e != nil {
				logEntry.Warn("cannot send response", zap.Error(e), zap.Binary("wire", wire))
			} else {
				logEntry.Debug("sent response")
			}
		}
	}
}

func pfcpDispatch(ctx context.Context, logEntry *zap.Logger, msg message.Message) (rsp message.Message, e error) {
	switch req := msg.(type) {
	case *message.HeartbeatRequest:
		return handleHeartbeat(req)
	case *message.AssociationSetupRequest:
		return handleAssoc(req, logEntry)
	case *message.SessionEstablishmentRequest:
		return handleSessEstab(ctx, req)
	case *message.SessionModificationRequest:
		return handleSessMod(ctx, req)
	case *message.SessionDeletionRequest:
		return handleSessDel(ctx, req)
	}
	return nil, errors.New("unhandled message type")
}

func handleHeartbeat(req *message.HeartbeatRequest) (rsp message.Message, e error) {
	return message.NewHeartbeatResponse(
		req.SequenceNumber,
		upfCfg.RecoveryTimestamp,
	), nil
}

func handleAssoc(req *message.AssociationSetupRequest, logEntry *zap.Logger) (rsp message.Message, e error) {
	logEntry.Info("association setup with SMF")
	return message.NewAssociationSetupResponse(
		req.SequenceNumber,
		upfCfg.UpfNodeID,
		ie.NewCause(ie.CauseRequestAccepted),
		upfCfg.RecoveryTimestamp,
	), nil
}
