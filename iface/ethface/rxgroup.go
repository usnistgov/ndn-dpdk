package ethface

import (
	"fmt"
)

// Ethernet RxGroup implementation.
type iRxgStarter interface {
	// Name of RxGroup implementation.
	String() string

	// Start the RxGroup implementation on a Port.
	// This function should create RxGroup(s) and EthFaces.
	Start(port *Port, cfg PortConfig) error
}

var rxgStarters = []iRxgStarter{rxFlowStarter{}, rxTableStarter{}}

type rxgStartError struct {
	Name  string
	Error error
}

type rxgStartErrors []rxgStartError

func (list rxgStartErrors) Error() (s string) {
	for i, e := range list {
		if i > 0 {
			s += "; "
		}
		s += fmt.Sprintf("%s: %s", e.Name, e.Error)
	}
	return s
}
