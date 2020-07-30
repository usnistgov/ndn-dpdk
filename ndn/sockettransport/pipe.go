package sockettransport

import (
	"net"
)

// Pipe creates a pair of transports connected via net.Pipe().
func Pipe(cfg Config) (trA, trB Transport, e error) {
	connA, connB := net.Pipe()

	trA, e = New(connA, cfg)
	if e != nil {
		return nil, nil, e
	}

	trB, e = New(connB, cfg)
	if e != nil {
		return nil, nil, e
	}

	return
}
