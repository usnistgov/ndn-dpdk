package ethport

import "fmt"

type rxImpl interface {
	fmt.Stringer
	Init(port *Port) error
	Start(face *Face) error
	Stop(face *Face) error
	Close(port *Port) error
}
