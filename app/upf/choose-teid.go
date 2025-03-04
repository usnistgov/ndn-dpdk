package upf

import (
	"maps"
	"math"
	"math/rand/v2"

	"github.com/usnistgov/ndn-dpdk/core/uintalloc"
	"github.com/wmnsk/go-pfcp/ie"
)

// SessionTeids contains TEIDs chosen within a PDU session.
//
//	Key: CHOOSE ID below 256; sequential since 256.
//	Value: TEID.
type SessionTeids map[uint16]uint32

// TeidChooser handles F-TEID allocation in the UP function.
type TeidChooser struct {
	MinTeid, MaxTeid uint32 // assignable TEID range

	inUse map[uint32]bool // TEIDs currently in use
}

func (tch *TeidChooser) randTeid() uint32 {
	return tch.MinTeid + rand.Uint32N(tch.MaxTeid-tch.MinTeid)
}

// Alloc assigns a TEID to a PDR.
//
//	fTEID: must have CH flag.
func (tch *TeidChooser) Alloc(fTEID *ie.FTEIDFields, sct SessionTeids) (teid uint32) {
	if fTEID.HasChID() {
		if teid, ok := sct[uint16(fTEID.ChooseID)]; ok {
			return teid
		}
	}

	teid = uintalloc.Alloc(tch.inUse, tch.randTeid)
	tch.inUse[teid] = true

	if fTEID.HasChID() {
		sct[uint16(fTEID.ChooseID)] = teid
	} else {
		sct[256+uint16(len(sct))] = teid
	}
	return teid
}

// Free releases TEIDs related to a session.
func (tch *TeidChooser) Free(sct SessionTeids) {
	for teid := range maps.Values(sct) {
		delete(tch.inUse, teid)
	}
}

// NewTeidChooser constructs TeidChooser.
func NewTeidChooser() *TeidChooser {
	return &TeidChooser{
		MaxTeid: math.MaxUint32,
		inUse:   map[uint32]bool{},
	}
}
