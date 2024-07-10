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
	sessHaveDlQERID
	sessHaveRemoteIP
	sessHaveInnerRemoteIP
	sessHaveAll = 0b1111111
)

// Session represents a PFCP session.
type Session struct {
	loc     SessionLocatorFields
	dlQERID uint32
	have    uint32
}

// EstablishmentRequest handles a SessionEstablishmentRequest message.
func (sess *Session) EstablishmentRequest(req *message.SessionEstablishmentRequest) error {
	return sess.emRequest(req.CreatePDR, req.CreateFAR, req.CreateQER)
}

// ModificationRequest handles a SessionModificationRequest message.
func (sess *Session) ModificationRequest(req *message.SessionModificationRequest) error {
	return sess.emRequest(req.CreatePDR, req.CreateFAR, req.CreateQER)
}

// emRequest handles a SessionEstablishmentRequest or SessionModificationRequest message.
func (sess *Session) emRequest(createPDR, createFAR, createQER []*ie.IE) error {
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
	qfi, e := findIE(ie.QFI).Within(pdi.PDI()).QFI()
	if e != nil {
		return fmt.Errorf("QFI: %w", e)
	}

	sess.loc.UlTEID, sess.loc.UlQFI = fTEID.TEID, qfi
	sess.have |= sessHaveUlTEID | sessHaveUlQFI
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

// createFARAccess handles a CreateFAR IE with DestinationInterface=access.
func (sess *Session) createFARAccess(far *ie.IE) error {
	ohc, e := findIE(ie.OuterHeaderCreation).Within(far.ForwardingParameters()).OuterHeaderCreation()
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
	if sess.have&sessHaveDlQERID == 0 || qerID != sess.dlQERID {
		return nil
	}

	sess.loc.DlQFI, e = qer.QFI()
	if e != nil {
		return fmt.Errorf("QFI: %w", e)
	}
	sess.have |= sessHaveDlQFI
	return nil
}

// LocatorFields returns GTP-U locator fields extracted from PFCP session.
// ok indicates whether the locator is valid.
func (sess Session) LocatorFields() (loc SessionLocatorFields, ok bool) {
	return sess.loc, sess.have == sessHaveAll
}
