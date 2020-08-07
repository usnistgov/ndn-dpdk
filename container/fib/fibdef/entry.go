package fibdef

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// EntryBody contains logical FIB entry contents except name.
type EntryBody struct {
	Nexthops []iface.ID `json:"nexthops"`
	Strategy int        `json:"strategy"`
}

// Equals determines whether two EntryBody records have the same values.
func (body EntryBody) Equals(other EntryBody) bool {
	if body.Strategy != other.Strategy || len(body.Nexthops) != len(other.Nexthops) {
		return false
	}
	for i, n := range body.Nexthops {
		if n != other.Nexthops[i] {
			return false
		}
	}
	return true
}

// Entry contains logical FIB entry contents.
type Entry struct {
	EntryBody
	Name ndn.Name `json:"name"`
}

// Validate checks entry fields.
func (entry *Entry) Validate() error {
	if entry.Name.Length() > MaxNameLength {
		return ErrNameTooLong
	}
	if len(entry.Nexthops) < 1 || len(entry.Nexthops) > MaxNexthops {
		return ErrNexthops
	}
	if entry.Strategy == 0 {
		return ErrStrategy
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
