package socketface

import (
	"fmt"
)

// Dial opens an L3Face over a socket using a default Dialer.
func Dial(network, local, remote string) (face *SocketFace, e error) {
	return Dialer{}.Dial(network, local, remote)
}

// Dialer contains settings for a created socket face.
type Dialer struct {
	Config
}

// Dial opens an L3Face over a socket, according to the configuration in the Dialer.
func (dialer Dialer) Dial(network, local, remote string) (*SocketFace, error) {
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
