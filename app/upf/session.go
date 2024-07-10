package upf

import (
	"errors"
	"fmt"
	"net/netip"

	"github.com/wmnsk/go-pfcp/ie"
	"github.com/wmnsk/go-pfcp/message"
)

// SessionLocatorFields contains GTP-U locator fields extracted from PFCP session.
type SessionLocatorFields struct {
	UlTEID        uint32     `json:"ulTEID"`
	DlTEID        uint32     `json:"dlTEID"`
	UlQFI         uint8      `json:"ulQFI"`
	DlQFI         uint8      `json:"dlQFI"`
	RemoteIP      netip.Addr `json:"remoteIP"`
	InnerRemoteIP netip.Addr `json:"innerRemoteIP"`
}

const (
	sessHaveUlTEID = 1 << iota
	sessHaveDlTEID
	sessHaveUlQFI
	sessHaveDlQFI
	sessHaveUlQERID
	sessHaveDlQERID
	sessHaveRemoteIP
	sessHaveInnerRemoteIP
	sessHaveAll = 0b11111111
)

// Session represents a PFCP session.
type Session struct {
	loc              SessionLocatorFields
	ulQERID, dlQERID uint32
	have             uint32
}

// EstablishmentRequest handles a SessionEstablishmentRequest message.
func (sess *Session) EstablishmentRequest(req *message.SessionEstablishmentRequest) error {
	return sess.emRequest(req.CreatePDR, req.CreateFAR, nil, req.CreateQER)
}

// ModificationRequest handles a SessionModificationRequest message.
func (sess *Session) ModificationRequest(req *message.SessionModificationRequest) error {
	return sess.emRequest(req.CreatePDR, req.CreateFAR, req.UpdateFAR, req.CreateQER)
}

// emRequest handles a SessionEstablishmentRequest or SessionModificationRequest message.
func (sess *Session) emRequest(createPDR, createFAR, updateFAR, createQER []*ie.IE) error {
	var errs []error
	for i, pdr := range createPDR {
		if e := sess.createPDR(pdr); e != nil {
			errs = append(errs, fmt.Errorf("CreatePDR[%d]: %w", i, e))
		}
	}
	for i, far := range createFAR {
		if e := sess.createFAR(far); e != nil {
			errs = append(errs, fmt.Errorf("CreateFAR[%d]: %w", i, e))
		}
	}
	for i, far := range updateFAR {
		if e := sess.updateFAR(far); e != nil {
			errs = append(errs, fmt.Errorf("UpdateFAR[%d]: %w", i, e))
		}
	}
	for i, qer := range createQER {
		if e := sess.createQER(qer); e != nil {
			errs = append(errs, fmt.Errorf("CreateQER[%d]: %w", i, e))
		}
	}
	return errors.Join(errs...)
}

// createPDR handles a CreatePDR IE.
func (sess *Session) createPDR(pdr *ie.IE) error {
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

// createPDRAccess handles a CreatePDR IE with SourceInterface=access.
func (sess *Session) createPDRAccess(pdr *ie.IE) error {
	pdi := findIE(ie.PDI).Within(pdr.CreatePDR())
	fTEID, e := pdi.FTEID()
	if e != nil {
		return fmt.Errorf("FTEID: %w", e)
	}

	sess.loc.UlTEID = fTEID.TEID
	sess.have |= sessHaveUlTEID

	sess.ulQERID, e = pdr.QERID()
	if e != nil {
		return fmt.Errorf("QERID: %w", e)
	}
	sess.have |= sessHaveUlQERID

	return nil
}

// createPDRCore handles a CreatePDR IE with SourceInterface=core.
func (sess *Session) createPDRCore(pdr *ie.IE) error {
	ueIP, e := pdr.UEIPAddress()
	if e != nil {
		return fmt.Errorf("UEIPAddress: %w", e)
	}

	ip, ok := netip.AddrFromSlice(ueIP.IPv4Address)
	if !ok || !ip.Is4() {
		return fmt.Errorf("UEIPAddress is not IPv4")
	}

	sess.loc.InnerRemoteIP = ip
	sess.have |= sessHaveInnerRemoteIP

	sess.dlQERID, e = pdr.QERID()
	if e != nil {
		return fmt.Errorf("QERID: %w", e)
	}
	sess.have |= sessHaveDlQERID

	return nil
}

// createFAR handles a CreateFAR IE.
func (sess *Session) createFAR(far *ie.IE) error {
	fps, e := far.ForwardingParameters()
	if e != nil {
		return fmt.Errorf("ForwardingParameters: %w", e)
	}
	return sess.cuFAR(fps)
}

// updateFAR handles an UpdateFAR IE.
func (sess *Session) updateFAR(far *ie.IE) error {
	fps, e := far.UpdateForwardingParameters()
	if e != nil {
		return fmt.Errorf("UpdateForwardingParameters: %w", e)
	}
	return sess.cuFAR(fps)
}

// cuFAR handles a CreateFAR or UpdateFAR IE.
func (sess *Session) cuFAR(fps []*ie.IE) error {
	if len(fps) == 0 {
		return errors.New("ForwardingParameters or UpdateForwardingParameters empty")
	}

	di, e := findIE(ie.DestinationInterface).Within(fps, nil).DestinationInterface()
	if e != nil {
		return fmt.Errorf("DestinationInterface: %w", e)
	}

	switch di {
	case ie.DstInterfaceAccess:
		return sess.cuFARAccess(fps)
	case ie.DstInterfaceCore:
		return nil
	}
	return fmt.Errorf("DestinationInterface %d unknown", di)
}

// cuFARAccess handles a CreateFAR or UpdateFAR IE with DestinationInterface=access.
func (sess *Session) cuFARAccess(fps []*ie.IE) error {
	ohcFound := findIE(ie.OuterHeaderCreation).Within(fps, nil)
	if ohcFound.Type == 0 {
		return nil
	}

	ohc, e := ohcFound.OuterHeaderCreation()
	if e != nil {
		return fmt.Errorf("OuterHeaderCreation: %w", e)
	}

	sess.loc.DlTEID = ohc.TEID
	sess.loc.RemoteIP, _ = netip.AddrFromSlice(ohc.IPv4Address)
	sess.have |= sessHaveDlTEID | sessHaveRemoteIP
	return nil
}

// createQER handles a CreateQER IE.
func (sess *Session) createQER(qer *ie.IE) error {
	qerID, e := qer.QERID()
	if e != nil {
		return fmt.Errorf("QERID: %w", e)
	}

	for _, c := range []struct {
		MustHave   uint32
		MatchQERID uint32
		SetQFI     *uint8
		SetHave    uint32
	}{
		{sessHaveUlQERID, sess.ulQERID, &sess.loc.UlQFI, sessHaveUlQFI},
		{sessHaveDlQERID, sess.dlQERID, &sess.loc.DlQFI, sessHaveDlQFI},
	} {
		if sess.have&c.MustHave != 0 && qerID == c.MatchQERID {
			*c.SetQFI, e = qer.QFI()
			if e != nil {
				return fmt.Errorf("QFI: %w", e)
			}
			sess.have |= c.SetHave
		}
	}
	return nil
}

// LocatorFields returns GTP-U locator fields extracted from PFCP session.
// ok indicates whether the locator is valid.
func (sess Session) LocatorFields() (loc SessionLocatorFields, ok bool) {
	return sess.loc, sess.have == sessHaveAll
}
