package nfdmgmt

import (
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

var (
	verbRibRegister   = ndn.ParseName("/rib/register")
	verbRibUnregister = ndn.ParseName("/rib/unregister")
)

func encodeRibCommonParameters(name ndn.Name, faceID int, origin int) (a []tlv.Fielder) {
	a = append(a, name)
	if faceID != 0 {
		a = append(a, tlv.TLVNNI(TtFaceID, faceID))
	}
	a = append(a, tlv.TLVNNI(TtOrigin, origin))
	return a
}

// RibRegisterCommand is a NFD command to register a route.
type RibRegisterCommand struct {
	Name      ndn.Name                `json:"name"`
	FaceID    int                     `json:"faceID,omitempty"`
	Origin    int                     `json:"origin"`
	Cost      int                     `json:"cost"`
	NoInherit bool                    `json:"noInherit"`
	Capture   bool                    `json:"capture"`
	Expires   nnduration.Milliseconds `json:"expires,omitempty"`
}

var _ ControlCommand = RibRegisterCommand{}

// Verb returns "rib/register".
func (RibRegisterCommand) Verb() []ndn.NameComponent {
	return verbRibRegister
}

// Parameters encodes ControlParameters.
func (cmd RibRegisterCommand) Parameters() (a []tlv.Fielder) {
	a = encodeRibCommonParameters(cmd.Name, cmd.FaceID, cmd.Origin)
	a = append(a, tlv.TLVNNI(TtCost, cmd.Cost))

	flags := 0
	if !cmd.NoInherit {
		flags |= 1
	}
	if cmd.Capture {
		flags |= 2
	}
	a = append(a, tlv.TLVNNI(TtFlags, flags))

	if cmd.Expires > 0 {
		a = append(a, tlv.TLVNNI(TtExpirationPeriod, cmd.Expires))
	}

	return a
}

// RibUnregisterCommand is a NFD command to unregister a route.
type RibUnregisterCommand struct {
	Name   ndn.Name `json:"name"`
	FaceID int      `json:"faceID,omitempty"`
	Origin int      `json:"origin"`
}

var _ ControlCommand = RibUnregisterCommand{}

// Verb returns "rib/unregister".
func (RibUnregisterCommand) Verb() []ndn.NameComponent {
	return verbRibUnregister
}

// Parameters encodes ControlParameters.
func (cmd RibUnregisterCommand) Parameters() (a []tlv.Fielder) {
	a = encodeRibCommonParameters(cmd.Name, cmd.FaceID, cmd.Origin)
	return a
}
