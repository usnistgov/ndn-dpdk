// Package upf provides a 5G User Plane Function that converts PFCP sessions to GTP-U faces.
package upf

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/wmnsk/go-pfcp/ie"
	"github.com/wmnsk/go-pfcp/message"
	"go.uber.org/zap"
)

var logger = logging.New("upf")

// UPF represents a User Plane Function.
type UPF struct {
	st     *SessionTable
	params UpfParams
}

// Listen listens for PFCP messages.
func (upf *UPF) Listen(ctx context.Context) error {
	addr := netip.AddrPortFrom(upf.params.UpfN4, 8805)
	conn, e := net.ListenUDP("udp", net.UDPAddrFromAddrPort(addr))
	if e != nil {
		return e
	}
	defer conn.Close()

	buf := make([]byte, 9000)
	for {
		n, raddr, e := conn.ReadFromUDPAddrPort(buf)
		if e != nil {
			return e
		}

		if raddr.Addr() != upf.params.SmfN4 {
			logger.Info("message not from SMF N4 address", zap.Stringer("smf", raddr))
			continue
		}

		wire := buf[:n]
		req, e := message.Parse(wire)
		if e != nil {
			logger.Warn("cannot parse message", zap.Error(e), zap.Binary("wire", wire))
			continue
		}
		logEntry := logger.With(zap.String("req-type", req.MessageTypeName()), zap.Uint32("req-seq", req.Sequence()))

		rsp, e := upf.ServePFCP(ctx, req)
		if e != nil {
			logEntry.Warn("cannot handle message", zap.Error(e))
		} else if rsp != nil {
			logEntry = logEntry.With(zap.String("rsp-type", rsp.MessageTypeName()))
			wire := make([]byte, rsp.MarshalLen())
			if e := rsp.MarshalTo(wire); e != nil {
				logEntry.Warn("cannot encode response", zap.Error(e))
			} else if _, e = conn.WriteToUDPAddrPort(wire, raddr); e != nil {
				logEntry.Warn("cannot send response", zap.Error(e), zap.Binary("wire", wire))
			} else {
				logEntry.Debug("sent response")
			}
		}
	}
}

// ServePFCP handles a PFCP message.
//
//	rsp: the response message, may be nil.
func (upf *UPF) ServePFCP(ctx context.Context, req message.Message) (rsp message.Message, e error) {
	switch req := req.(type) {
	case *message.HeartbeatRequest:
		return upf.HeartbeatRequest(ctx, req)
	case *message.AssociationSetupRequest:
		return upf.AssociationSetupRequest(ctx, req)
	case *message.SessionEstablishmentRequest:
		return upf.SessionEstablishmentRequest(ctx, req)
	case *message.SessionModificationRequest:
		return upf.SessionModificationRequest(ctx, req)
	case *message.SessionDeletionRequest:
		return upf.SessionDeletionRequest(ctx, req)
	}
	return nil, errors.New("unhandled message type")
}

// HeartbeatRequest handles a HeartbeatRequest message.
func (upf *UPF) HeartbeatRequest(ctx context.Context, req *message.HeartbeatRequest) (rsp *message.HeartbeatResponse, e error) {
	return message.NewHeartbeatResponse(
		req.SequenceNumber,
		upf.params.RecoveryTimestamp,
	), nil
}

// AssociationSetupRequest handles an AssociationSetupRequest message.
func (upf *UPF) AssociationSetupRequest(ctx context.Context, req *message.AssociationSetupRequest) (rsp *message.AssociationSetupResponse, e error) {
	nodeID, e := req.NodeID.NodeID()
	if e != nil {
		return nil, fmt.Errorf("NodeID: %w", e)
	}
	logger.Info("association setup", zap.String("cp-node", nodeID))
	return message.NewAssociationSetupResponse(
		req.SequenceNumber,
		upf.params.UpfNodeID,
		ie.NewCause(ie.CauseRequestAccepted),
		upf.params.RecoveryTimestamp,
	), nil
}

// SessionEstablishmentRequest handles a SessionEstablishmentRequest message.
func (upf *UPF) SessionEstablishmentRequest(ctx context.Context, req *message.SessionEstablishmentRequest) (rsp *message.SessionEstablishmentResponse, e error) {
	sess, e := upf.st.EstablishmentRequest(ctx, req)
	if sess == nil || e != nil {
		return nil, e
	}
	return message.NewSessionEstablishmentResponse(
		0, 0, sess.CpSEID, req.SequenceNumber, 0,
		upf.params.UpfNodeID,
		ie.NewCause(ie.CauseRequestAccepted),
		ie.NewFSEID(sess.UpSEID, upf.params.UpfN4.AsSlice(), nil),
	), nil
}

// SessionModificationRequest handles a SessionModificationRequest message.
func (upf *UPF) SessionModificationRequest(ctx context.Context, req *message.SessionModificationRequest) (rsp *message.SessionModificationResponse, e error) {
	sess, e := upf.st.ModificationRequest(ctx, req)
	if sess == nil {
		return nil, e
	}
	return message.NewSessionModificationResponse(
		0, 0, sess.CpSEID, req.SequenceNumber, 0,
		ie.NewCause(ie.CauseRequestAccepted),
	), e
}

// SessionDeletionRequest handles a SessionDeletionRequest message.
func (upf *UPF) SessionDeletionRequest(ctx context.Context, req *message.SessionDeletionRequest) (rsp *message.SessionDeletionResponse, e error) {
	sess, e := upf.st.DeletionRequest(ctx, req)
	if sess == nil {
		return nil, e
	}
	return message.NewSessionDeletionResponse(
		0, 0, sess.CpSEID, req.SequenceNumber, 0,
		ie.NewCause(ie.CauseRequestAccepted),
	), e
}

// NewUPF constructs UPF.
func NewUPF(
	params UpfParams,
	createFace func(ctx context.Context, loc any) (id string, e error),
	destroyFace func(ctx context.Context, id string) error,
) *UPF {
	return &UPF{
		st: NewSessionTable(
			func(ctx context.Context, sloc SessionLocatorFields) (id string, e error) {
				loc, e := params.MakeLocator(sloc)
				if e != nil {
					return "", e
				}
				return createFace(ctx, loc)
			},
			destroyFace,
		),
		params: params,
	}
}
