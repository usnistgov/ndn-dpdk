package fibdef

import (
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// UpdateAction indicates entry update action.
type UpdateAction int

// UpdateAction values.
const (
	ActInsert UpdateAction = iota + 1
	ActReplace
	ActErase
)

// RealUpdate represents a real entry update command.
type RealUpdate struct {
	EntryBody
	Name     ndn.Name
	Action   UpdateAction
	WithVirt *VirtUpdate
}

// VirtUpdate represents a virtual entry update command.
type VirtUpdate struct {
	Name    ndn.Name
	Action  UpdateAction
	HasReal bool
	Height  int
}

// Update represents a tree update command.
type Update interface {
	Real() *RealUpdate
	Virt() *VirtUpdate
	Commit()
	Revert()
}
