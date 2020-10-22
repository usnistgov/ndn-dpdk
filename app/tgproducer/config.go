package tgproducer

import (
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// Config contains traffic generator producer configuration.
type Config struct {
	RxQueue  iface.PktQueueConfig `json:"rxQueue,omitempty"`
	Patterns []Pattern            `json:"patterns"`
	// true: respond Nacks to unmatched Interests
	// false: drop unmatched Interests
	Nack bool `json:"nack,omitempty"`
}

// Pattern configures how the producer replies to Interests under a name prefix.
type Pattern struct {
	Prefix  ndn.Name `json:"prefix"`
	Replies []Reply  `json:"replies"`
}

// Reply configures how the producer replies to the Interest.
type Reply struct {
	Weight int `json:"weight,omitempty"` // weight of random choice, minimum/default is 1

	Suffix          ndn.Name                `json:"suffix,omitempty"` // suffix to append to Interest name
	FreshnessPeriod nnduration.Milliseconds `json:"freshnessPeriod,omitempty"`
	PayloadLen      int                     `json:"payloadLen,omitempty"`

	Nack uint8 `json:"nack,omitempty"` // if not NackNone, reply with Nack instead of Data

	Timeout bool `json:"timeout,omitempty"` // if true, drop the Interest instead of sending Data
}
