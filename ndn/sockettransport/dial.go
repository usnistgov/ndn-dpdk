package sockettransport

import (
	"fmt"
)

// Dial opens a socket transport using a default Dialer.
func Dial(network, local, remote string) (Transport, error) {
	return Dialer{}.Dial(network, local, remote)
}

// Dialer contains settings for Dial.
type Dialer struct {
	Config
}

// Dial opens a socket transport, according to the configuration in the Dialer.
func (dialer Dialer) Dial(network, local, remote string) (Transport, error) {
	dialer.Config.applyDefaults()

	impl, ok := implByNetwork[network]
	if !ok {
		return nil, fmt.Errorf("unknown network %s", network)
	}

	conn, e := impl.Dial(network, local, remote)
	if e != nil {
		return nil, e
	}

	return New(conn, dialer.Config)
}
