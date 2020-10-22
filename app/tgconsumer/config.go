package tgconsumer

import (
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// Config contains traffic generator consumer configuration.
type Config struct {
	RxQueue  iface.PktQueueConfig   `json:"rxQueue,omitempty"`
	Patterns []Pattern              `json:"patterns"`
	Interval nnduration.Nanoseconds `json:"interval"`
}

// Pattern configures how the consumer generates a sequence of Interests.
type Pattern struct {
	Weight int `json:"weight,omitempty"` // weight of random choice, minimum/default is 1

	Prefix           ndn.Name                `json:"prefix"`
	CanBePrefix      bool                    `json:"canBePrefix,omitempty"`
	MustBeFresh      bool                    `json:"mustBeFresh,omitempty"`
	InterestLifetime nnduration.Milliseconds `json:"interestLifetime,omitempty"`
	HopLimit         ndn.HopLimit            `json:"hopLimit,omitempty"`

	// If non-zero, request cached Data. This must appear after a pattern without SeqNumOffset.
	// The client derives sequece number by subtracting SeqNumOffset from the previous pattern's
	// sequence number. Sufficient CS capacity is necessary for Data to actually come from CS.
	SeqNumOffset int `json:"seqNumOffset,omitempty"`
}
