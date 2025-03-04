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
	spHaveUlTEID = 1 << iota
	spHaveDlTEID
	spHaveUlQFI
	spHaveDlQFI
	spHaveUlQERID
	spHaveDlQERID
	spHaveRemoteIP
	spHaveInnerRemoteIP
	spHaveNeeded = spHaveUlTEID | spHaveDlTEID | spHaveUlQFI | spHaveDlQFI | spHaveRemoteIP | spHaveInnerRemoteIP
)

// SessionParser parses PFCP messages to construct GTP-U face locator.
type SessionParser struct {
	// Handle F-TEID with CH flag, returns updated F-TEID for CreatedPDR.
	ChooseTeid func(fTEID *ie.FTEIDFields) *ie.FTEIDFields

	loc              SessionLocatorFields
	ulQERID, dlQERID uint32
	have             uint32
}

// EstablishmentRequest handles a SessionEstablishmentRequest message.
func (sp *SessionParser) EstablishmentRequest(req *message.SessionEstablishmentRequest, rspIEs []*ie.IE) ([]*ie.IE, error) {
	return sp.emRequest(req.CreatePDR, req.CreateFAR, nil, req.CreateQER, rspIEs)
}

// ModificationRequest handles a SessionModificationRequest message.
func (sp *SessionParser) ModificationRequest(req *message.SessionModificationRequest, rspIEs []*ie.IE) ([]*ie.IE, error) {
	return sp.emRequest(req.CreatePDR, req.CreateFAR, req.UpdateFAR, req.CreateQER, rspIEs)
}

// emRequest handles a SessionEstablishmentRequest or SessionModificationRequest message.
func (sp *SessionParser) emRequest(createPDR, createFAR, updateFAR, createQER []*ie.IE, rspIEs []*ie.IE) (rspIEsRet []*ie.IE, e error) {
	var errs []error
	for i, pdr := range createPDR {
		if rspIEs, e = sp.createPDR(pdr, rspIEs); e != nil {
			errs = append(errs, fmt.Errorf("CreatePDR[%d]: %w", i, e))
		}
	}
	for i, far := range createFAR {
		if e = sp.createFAR(far); e != nil {
			errs = append(errs, fmt.Errorf("CreateFAR[%d]: %w", i, e))
		}
	}
	for i, far := range updateFAR {
		if e = sp.updateFAR(far); e != nil {
			errs = append(errs, fmt.Errorf("UpdateFAR[%d]: %w", i, e))
		}
	}
	for i, qer := range createQER {
		if e = sp.createQER(qer); e != nil {
			errs = append(errs, fmt.Errorf("CreateQER[%d]: %w", i, e))
		}
	}
	return rspIEs, errors.Join(errs...)
}

// createPDR handles a CreatePDR IE.
func (sp *SessionParser) createPDR(pdr *ie.IE, rspIEs []*ie.IE) ([]*ie.IE, error) {
	si, e := pdr.SourceInterface()
	if e != nil {
		return rspIEs, fmt.Errorf("SourceInterface: %w", e)
	}

	pdrID, e := pdr.PDRID()
	if e != nil {
		return rspIEs, fmt.Errorf("PDRID: %w", e)
	}
	createdPdr := []*ie.IE{
		ie.NewPDRID(pdrID),
	}

	isAccess := false
	switch si {
	case ie.SrcInterfaceAccess:
		isAccess = true
		fallthrough
	case ie.SrcInterfaceCPFunction:
		fTEID, e := sp.createPDRWithFTEID(pdr, isAccess)
		if e != nil {
			return rspIEs, e
		}
		createdPdr = append(createdPdr, encodeFTEID(*fTEID))
	case ie.SrcInterfaceCore:
		return rspIEs, sp.createPDRCore(pdr)
	default:
		return rspIEs, fmt.Errorf("SourceInterface %d unknown", si)
	}

	rspIEs = append(rspIEs, ie.NewCreatedPDR(createdPdr...))
	return rspIEs, nil
}

// createPDRWithFTEID handles a CreatePDR IE expected to contain PDI.
func (sp *SessionParser) createPDRWithFTEID(pdr *ie.IE, isAccess bool) (*ie.FTEIDFields, error) {
	pdi := FindIE(ie.PDI).Within(pdr.CreatePDR())
	fTEID, e := pdi.FTEID()
	if e != nil {
		return nil, fmt.Errorf("FTEID: %w", e)
	}
	if !fTEID.HasIPv4() {
		return nil, fmt.Errorf("FTEID without IPv4 flag is not supported")
	}

	if fTEID.HasCh() {
		fTEID = sp.ChooseTeid(fTEID)
	}

	sp.loc.UlTEID = fTEID.TEID
	sp.have |= spHaveUlTEID

	sp.ulQERID, e = pdr.QERID()
	if e == nil {
		sp.have |= spHaveUlQERID
	} else if errors.Is(e, ie.ErrIENotFound) {
		// OAI-CN5G-SMF v2.0.1 does not send CreateQER, but QFI is available in the PDI.
		// UlQFI and DlQFI are assumed to be the same.
		sp.loc.UlQFI, e = FindIE(ie.QFI).Within(pdi.PDI()).QFI()
		if e != nil {
			return nil, fmt.Errorf("QFI: %w", e)
		}
		sp.loc.DlQFI = sp.loc.UlQFI
		sp.have |= spHaveUlQFI | spHaveDlQFI
	} else {
		return nil, fmt.Errorf("QERID: %w", e)
	}

	return fTEID, nil
}

// createPDRCore handles a CreatePDR IE with SourceInterface=core.
func (sp *SessionParser) createPDRCore(pdr *ie.IE) error {
	ueIP, e := pdr.UEIPAddress()
	if e != nil {
		return fmt.Errorf("UEIPAddress: %w", e)
	}

	ip, ok := netip.AddrFromSlice(ueIP.IPv4Address)
	if !ok || !ip.Is4() {
		return fmt.Errorf("UEIPAddress is not IPv4")
	}

	sp.loc.InnerRemoteIP = ip
	sp.have |= spHaveInnerRemoteIP

	sp.dlQERID, e = pdr.QERID()
	if e == nil {
		sp.have |= spHaveDlQERID
	} else if !errors.Is(e, ie.ErrIENotFound) {
		return fmt.Errorf("QERID: %w", e)
	}

	return nil
}

// createFAR handles a CreateFAR IE.
func (sp *SessionParser) createFAR(far *ie.IE) error {
	if !far.HasFORW() {
		return nil
	}

	fps, e := far.ForwardingParameters()
	if e != nil {
		return fmt.Errorf("ForwardingParameters: %w", e)
	}
	return sp.cuFAR(fps)
}

// updateFAR handles an UpdateFAR IE.
func (sp *SessionParser) updateFAR(far *ie.IE) error {
	fps, e := far.UpdateForwardingParameters()
	if e != nil {
		return fmt.Errorf("UpdateForwardingParameters: %w", e)
	}
	return sp.cuFAR(fps)
}

// cuFAR handles a CreateFAR or UpdateFAR IE.
func (sp *SessionParser) cuFAR(fps []*ie.IE) error {
	if len(fps) == 0 {
		return errors.New("ForwardingParameters or UpdateForwardingParameters empty")
	}

	di, e := FindIE(ie.DestinationInterface).Within(fps, nil).DestinationInterface()
	if e != nil {
		return fmt.Errorf("DestinationInterface: %w", e)
	}

	switch di {
	case ie.DstInterfaceAccess:
		return sp.cuFARAccess(fps)
	case ie.DstInterfaceCore, ie.DstInterfaceCPFunction:
		return nil
	}
	return fmt.Errorf("DestinationInterface %d unknown", di)
}

// cuFARAccess handles a CreateFAR or UpdateFAR IE with DestinationInterface=access.
func (sp *SessionParser) cuFARAccess(fps []*ie.IE) error {
	ohcFound := FindIE(ie.OuterHeaderCreation).Within(fps, nil)
	if ohcFound.Type == 0 {
		return nil
	}

	ohc, e := ohcFound.OuterHeaderCreation()
	if e != nil {
		return fmt.Errorf("OuterHeaderCreation: %w", e)
	}

	sp.loc.DlTEID = ohc.TEID
	sp.loc.RemoteIP, _ = netip.AddrFromSlice(ohc.IPv4Address)
	sp.have |= spHaveDlTEID | spHaveRemoteIP
	return nil
}

// createQER handles a CreateQER IE.
func (sp *SessionParser) createQER(qer *ie.IE) error {
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
		{spHaveUlQERID, sp.ulQERID, &sp.loc.UlQFI, spHaveUlQFI},
		{spHaveDlQERID, sp.dlQERID, &sp.loc.DlQFI, spHaveDlQFI},
	} {
		if sp.have&c.MustHave != 0 && qerID == c.MatchQERID {
			*c.SetQFI, e = qer.QFI()
			if e != nil {
				return fmt.Errorf("QFI: %w", e)
			}
			sp.have |= c.SetHave
		}
	}
	return nil
}

// LocatorFields returns GTP-U locator fields extracted from PFCP session.
// ok indicates whether the locator is valid.
func (sp SessionParser) LocatorFields() (loc SessionLocatorFields, ok bool) {
	return sp.loc, sp.have&spHaveNeeded == spHaveNeeded
}
