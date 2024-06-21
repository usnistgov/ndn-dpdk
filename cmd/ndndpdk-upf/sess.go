package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/netip"

	"github.com/wmnsk/go-pfcp/ie"
	"github.com/wmnsk/go-pfcp/message"
	"go.uber.org/zap"
)

const (
	sessHaveUlTEID = 1 << iota
	sessHaveDlTEID
	sessHaveUlQFI
	sessHaveDlQFI
	sessHaveDlQERID
	sessHavePeer
	sessHaveUeIP
	sessHaveAll = 0b1111111
)

var pfcpSessTable = make(map[uint64]*pfcpSess)

type pfcpSess struct {
	cpSEID, upSEID uint64
	ulTEID, dlTEID uint32
	ulQFI, dlQFI   uint8
	dlQERID        uint32
	peer, ueIP     netip.Addr
	have           uint32
	FaceID         string
}

func (sess pfcpSess) decorateLogEntry(logger *zap.Logger) *zap.Logger {
	return logger.With(zap.Uint64("cp-seid", sess.cpSEID), zap.Uint64("up-seid", sess.upSEID))
}

func (sess *pfcpSess) HandleCreate(ctx context.Context, logEntry *zap.Logger, pdrs, fars, qers []*ie.IE) {
	logEntry = sess.decorateLogEntry(logEntry)

	for i, pdr := range pdrs {
		if e := sess.createPDR(pdr); e != nil {
			logEntry.Info("createPDR error", zap.Int("index", i), zap.Error(e))
		}
	}
	for i, far := range fars {
		if e := sess.createFAR(far); e != nil {
			logEntry.Info("createFAR error", zap.Int("index", i), zap.Error(e))
		}
	}
	for i, qer := range qers {
		if e := sess.createQER(qer); e != nil {
			logEntry.Info("createQER error", zap.Int("index", i), zap.Error(e))
		}
	}
	if sess.have != sessHaveAll {
		logEntry.Debug("waiting for more updates", zap.Uint32("have", sess.have))
		return
	}

	loc, e := upfCfg.MakeLocator(sess.ulTEID, sess.ulQFI, sess.dlTEID, sess.dlQFI, sess.peer, sess.ueIP)
	if e != nil {
		logEntry.Warn("cannot construct locator", zap.Error(e))
		return
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
		logEntry.Warn("cannot create face", zap.Any("locator", loc), zap.Error(e))
		return
	}
	sess.FaceID = reply.ID
	logEntry.Info("face created", zap.Any("locator", loc), zap.String("face-id", sess.FaceID))
}

func (sess *pfcpSess) createPDR(pdr *ie.IE) error {
	si, e := pdr.SourceInterface()
	if e != nil {
		return fmt.Errorf("SourceInterface: %w", e)
	}

	switch si {
	case ie.SrcInterfaceAccess:
		return sess.createPDRAccess(pdr)
	case ie.SrcInterfaceCore:
		return sess.createPDRCore(pdr)
	}
	return fmt.Errorf("SourceInterface %d unknown", si)
}

func (sess *pfcpSess) createPDRAccess(pdr *ie.IE) error {
	pdi := findIE(ie.PDI).Within(pdr.CreatePDR())
	fTEID, e := pdi.FTEID()
	if e != nil {
		return fmt.Errorf("FTEID: %w", e)
	}
	qfi, e := findIE(ie.QFI).Within(pdi.PDI()).QFI()
	if e != nil {
		return fmt.Errorf("QFI: %w", e)
	}

	sess.ulTEID, sess.ulQFI = fTEID.TEID, qfi
	sess.have |= sessHaveUlTEID | sessHaveUlQFI
	return nil
}

func (sess *pfcpSess) createPDRCore(pdr *ie.IE) error {
	ueIP, e := pdr.UEIPAddress()
	if e != nil {
		return fmt.Errorf("UEIPAddress: %w", e)
	}

	ip, ok := netip.AddrFromSlice(ueIP.IPv4Address)
	if !ok || !ip.Is4() {
		return fmt.Errorf("UEIPAddress is not IPv4")
	}

	sess.ueIP = ip
	sess.have |= sessHaveUeIP

	sess.dlQERID, e = pdr.QERID()
	if e != nil {
		return fmt.Errorf("QERID: %w", e)
	}
	sess.have |= sessHaveDlQERID

	return nil
}

func (sess *pfcpSess) createFAR(far *ie.IE) error {
	fps, e := far.ForwardingParameters()
	if e != nil {
		return fmt.Errorf("ForwardingParameters: %w", e)
	}
	if len(fps) == 0 {
		return errors.New("ForwardingParameters empty")
	}

	di, e := findIE(ie.DestinationInterface).Within(far.ForwardingParameters()).DestinationInterface()
	if e != nil {
		return fmt.Errorf("DestinationInterface: %w", e)
	}

	switch di {
	case ie.DstInterfaceAccess:
		return sess.createFARAccess(far)
	case ie.DstInterfaceCore:
		return nil
	}
	return fmt.Errorf("DestinationInterface %d unknown", di)
}

func (sess *pfcpSess) createFARAccess(far *ie.IE) error {
	ohc, e := findIE(ie.OuterHeaderCreation).Within(far.ForwardingParameters()).OuterHeaderCreation()
	if e != nil {
		return fmt.Errorf("OuterHeaderCreation: %w", e)
	}

	sess.dlTEID = ohc.TEID
	sess.peer, _ = netip.AddrFromSlice(ohc.IPv4Address)
	sess.have |= sessHaveDlTEID | sessHavePeer
	return nil
}

func (sess *pfcpSess) createQER(qer *ie.IE) error {
	qerID, e := qer.QERID()
	if e != nil {
		return fmt.Errorf("QERID: %w", e)
	}
	if sess.have&sessHaveDlQERID == 0 || qerID != sess.dlQERID {
		return nil
	}

	sess.dlQFI, e = qer.QFI()
	if e != nil {
		return fmt.Errorf("QFI: %w", e)
	}
	sess.have |= sessHaveDlQFI
	return nil
}

func (sess *pfcpSess) Delete(ctx context.Context, logEntry *zap.Logger) {
	logEntry = sess.decorateLogEntry(logEntry).With(zap.String("face-id", sess.FaceID))
	if sess.FaceID == "" {
		logEntry.Debug("face does not exist, will not delete")
		return
	}

	deleted, e := client.Delete(ctx, sess.FaceID)
	if e != nil {
		logEntry.Warn("cannot delete face", zap.Error(e))
	} else {
		logEntry.Info("face deleted", zap.Bool("deleted", deleted))
	}
}

func handleSessEstab(ctx context.Context, logEntry *zap.Logger, req *message.SessionEstablishmentRequest) (rsp message.Message, e error) {
	cpfSEID, e := req.CPFSEID.FSEID()
	if e != nil {
		return nil, fmt.Errorf("CP F-SEID: %w", e)
	}

	sess := &pfcpSess{
		cpSEID: cpfSEID.SEID,
	}
	for sess.upSEID == 0 || pfcpSessTable[sess.upSEID] != nil {
		sess.upSEID = rand.Uint64()
	}
	pfcpSessTable[sess.upSEID] = sess

	sess.HandleCreate(ctx, logEntry, req.CreatePDR, req.CreateFAR, req.CreateQER)
	return message.NewSessionEstablishmentResponse(
		0, 0, sess.cpSEID, req.SequenceNumber, 0,
		upfCfg.UpfNodeID,
		ie.NewCause(ie.CauseRequestAccepted),
		ie.NewFSEID(sess.upSEID, upfCfg.UpfN4.AsSlice(), nil),
	), nil
}

func handleSessMod(ctx context.Context, logEntry *zap.Logger, req *message.SessionModificationRequest) (rsp message.Message, e error) {
	sess := pfcpSessTable[req.SEID()]
	if sess == nil {
		return message.NewSessionModificationResponse(
			0, 0, 0, req.SequenceNumber, 0,
			ie.NewCause(ie.CauseSessionContextNotFound),
		), nil
	}

	sess.HandleCreate(ctx, logEntry, req.CreatePDR, req.CreateFAR, req.CreateQER)
	return message.NewSessionModificationResponse(
		0, 0, sess.cpSEID, req.SequenceNumber, 0,
		ie.NewCause(ie.CauseRequestAccepted),
	), nil
}

func handleSessDel(ctx context.Context, logEntry *zap.Logger, req *message.SessionDeletionRequest) (rsp message.Message, e error) {
	sess := pfcpSessTable[req.SEID()]
	if sess == nil {
		return message.NewSessionDeletionResponse(
			0, 0, 0, req.SequenceNumber, 0,
			ie.NewCause(ie.CauseSessionContextNotFound),
		), nil
	}

	sess.Delete(ctx, logEntry)

	return message.NewSessionDeletionResponse(
		0, 0, sess.cpSEID, req.SequenceNumber, 0,
		ie.NewCause(ie.CauseRequestAccepted),
	), nil
}
