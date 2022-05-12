package ethport

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/iface"
)

type rxImpl interface {
	fmt.Stringer
	List(port *Port) []iface.RxGroup
	Init(port *Port) error
	Start(face *Face) error
	Stop(face *Face) error
	Close(port *Port) error
}
