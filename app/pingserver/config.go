package pingserver

import (
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// Server config.
type Config struct {
	RxQueue  iface.PktQueueConfig
	Patterns []Pattern // traffic patterns
	Nack     bool      // whether to respond Nacks to unmatched Interests
}

// Server pattern definition.
type Pattern struct {
	Prefix  ndn.Name // name prefix
	Replies []Reply  // reply settings
}

// Server reply definition.
type Reply struct {
	Weight int // weight of random choice, minimum is 1

	Suffix          ndn.Name                // suffix to append to Interest name
	FreshnessPeriod nnduration.Milliseconds // FreshnessPeriod value
	PayloadLen      int                     // Content payload length

	Nack uint8 // if not NackNone, reply with Nack instead of Data

	Timeout bool // if true, drop the Interest instead of sending Data
}
