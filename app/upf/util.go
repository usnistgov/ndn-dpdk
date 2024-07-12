package upf

import "github.com/wmnsk/go-pfcp/ie"

// findIE searches for an IE within a grouped IE.
// Its value is the desired IE type.
type findIE uint16

// Within performs the search.
// Its parameters match existing functions that return grouped IE with error.
// If the second parameter indicates an error or the desired IE type is absent, returns zero IE.
func (typ findIE) Within(ies []*ie.IE, e error) *ie.IE {
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
