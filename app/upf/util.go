package upf

import (
	"iter"
	"slices"

	"github.com/bobg/seqs"
	"github.com/wmnsk/go-pfcp/ie"
)

// FindIE searches for an IE within a grouped IE.
// Its value is the desired IE type.
type FindIE uint16

// Within finds the first IE of the desired type.
//
// Its parameters match existing functions that return grouped IE with error.
// If the second parameter indicates an error or the desired IE type is absent, returns zero IE.
func (typ FindIE) Within(ies []*ie.IE, e error) *ie.IE {
	var zero ie.IE
	if e != nil {
		return &zero
	}

	for _, item := range ies {
		if item.Type == uint16(typ) {
			return item
		}
	}

	return &zero
}

// IterWithin iterates over all IEs of the desired type.
func (typ FindIE) IterWithin(ies []*ie.IE) iter.Seq[*ie.IE] {
	return seqs.Filter(slices.Values(ies), func(item *ie.IE) bool {
		return item.Type == uint16(typ)
	})
}

// encodeFTEID encodes F-TEID from fields.
func encodeFTEID(fTEID ie.FTEIDFields) *ie.IE {
	return ie.NewFTEID(fTEID.Flags, fTEID.TEID, fTEID.IPv4Address, fTEID.IPv6Address, fTEID.ChooseID)
}
