package l3

import "github.com/usnistgov/ndn-dpdk/ndn"

// ReadvertiseDestination represents a destination of name advertisement.
//
// Generally, a name advertised to a destination would cause Interests matching the name to come to the forwarder.
// This is also known as name registration.
type ReadvertiseDestination interface {
	Advertise(name ndn.Name) error

	Withdraw(name ndn.Name) error
}
