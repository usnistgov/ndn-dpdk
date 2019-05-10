package ethface

import (
	"fmt"
	"io"
)

// Ethernet RxGroup implementation.
type iImpl interface {
	fmt.Stringer
	io.Closer

	Init() error

	Start(face *EthFace) error

	Stop(face *EthFace) error
}

// var rxgStarters = []iRxgStarter{rxFlowStarter{}, rxTableStarter{}}

// type rxgStartError struct {
// 	Name  string
// 	Error error
// }

// type rxgStartErrors []rxgStartError

// func (list rxgStartErrors) Error() (s string) {
// 	for i, e := range list {
// 		if i > 0 {
// 			s += "; "
// 		}
// 		s += fmt.Sprintf("%s: %s", e.Name, e.Error)
// 	}
// 	return s
// }
