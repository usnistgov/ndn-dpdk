package fibdef

import (
	"errors"
	"fmt"

	"slices"

	"github.com/suzuki-shunsuke/go-dataeq/dataeq"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// EntryBody contains logical FIB entry contents except name.
type EntryBody struct {
	Nexthops []iface.ID     `json:"nexthops"`
	Strategy int            `json:"strategy"`
	Params   map[string]any `json:"params"`
}

// HasNextHop determines whether a nexthop face exists.
func (entry EntryBody) HasNextHop(id iface.ID) bool {
	return slices.Contains(entry.Nexthops, id)
}

// EntryBodyEquals determines whether two EntryBody records have the same values.
func EntryBodyEquals(lhs, rhs EntryBody) bool {
	if lhs.Strategy != rhs.Strategy || len(lhs.Nexthops) != len(rhs.Nexthops) {
		return false
	}
	for i, n := range lhs.Nexthops {
		if n != rhs.Nexthops[i] {
			return false
		}
	}
	if eq, e := dataeq.JSON.Equal(lhs.Params, rhs.Params); e != nil || !eq {
		return false
	}
	return true
}

// Entry contains logical FIB entry contents.
type Entry struct {
	EntryBody
	Name ndn.Name `json:"name"`
}

// Validate checks entry fields.
func (entry Entry) Validate() error {
	if entry.Name.Length() > MaxNameLength {
		return errors.New("FIB entry name too long")
	}
	if len(entry.Nexthops) < 1 || len(entry.Nexthops) > MaxNexthops {
		return fmt.Errorf("number of nexthops must be between 1 and %d", MaxNexthops)
	}
	if entry.Strategy == 0 {
		return errors.New("missing strategy")
	}
	return nil
}

// EntryCounters contains entry counters.
type EntryCounters struct {
	NRxInterests uint64 `json:"nRxInterests"`
	NRxData      uint64 `json:"nRxData"`
	NRxNacks     uint64 `json:"nRxNacks"`
	NTxInterests uint64 `json:"nTxInterests"`
}

func (cnt EntryCounters) String() string {
	return fmt.Sprintf("%dI %dD %dN %dO", cnt.NRxInterests, cnt.NRxData, cnt.NRxNacks, cnt.NTxInterests)
}
