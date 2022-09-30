package nfdmgmt

import (
	"crypto/rand"
	"errors"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// ControlCommand represents a NFD control command.
type ControlCommand interface {
	Verb() []ndn.NameComponent
	Parameters() []tlv.Fielder
}

// MakeCommandInterest constructs a control command Interest without signing.
func MakeCommandInterest(commandPrefix ndn.Name, cmd ControlCommand) ndn.Interest {
	name := commandPrefix.Append(cmd.Verb()...)
	name = append(name, ndn.NameComponentFrom(an.TtGenericNameComponent, tlv.TLVFrom(TtControlParameters, cmd.Parameters()...)))
	var sigNonce [8]byte
	rand.Read(sigNonce[:])
	return ndn.Interest{
		Name:        name,
		MustBeFresh: true,
		SigInfo: &ndn.SigInfo{
			Nonce: sigNonce[:],
			Time:  uint64(time.Now().UnixMilli()),
		},
	}
}

// ControlResponse represents a NFD control response.
type ControlResponse struct {
	StatusCode int
	StatusText string
	Body       []byte
}

func (cr *ControlResponse) UnmarshalTLV(typ uint32, value []byte) error {
	if typ != TtControlResponse {
		return tlv.ErrType
	}

	d := tlv.DecodingBuffer(value)

	de, e := d.Element()
	if e != nil || de.Type != TtStatusCode {
		goto FAIL
	}
	if cr.StatusCode = int(de.UnmarshalNNI(999, &e, tlv.ErrRange)); e != nil {
		goto FAIL
	}

	de, e = d.Element()
	if e != nil || de.Type != TtStatusText {
		goto FAIL
	}
	cr.StatusText = string(de.Value)

	cr.Body = d.Rest()
	return nil

FAIL:
	return errors.New("bad ControlResponse")
}
