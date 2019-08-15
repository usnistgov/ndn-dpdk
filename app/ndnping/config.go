package ndnping

import (
	"time"

	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

// Package initialization config.
type InitConfig struct {
	QueueCapacity int // input-client/server queue capacity, must be power of 2
}

// Per-face task config, consists of a client and/or a server.
type TaskConfig struct {
	Face   iface.LocatorWrapper // face locator for face creation
	Client *ClientConfig        // if not nil, create a client on the face
	Server *ServerConfig        // if not nil, create a server on the face
}

// Client config.
type ClientConfig struct {
	Patterns []ClientPattern // traffic patterns
	Interval time.Duration   // sending interval
}

// Client pattern defintion.
type ClientPattern struct {
	Weight int // weight of random choice, minimum is 1

	Prefix           *ndn.Name     // name prefix
	CanBePrefix      bool          // whether to set CanBePrefix
	MustBeFresh      bool          // whether to set MustBeFresh
	InterestLifetime time.Duration // InterestLifetime value, zero means default
	HopLimit         int           // HopLimit value, zero means default

	// If non-zero, request cached Data. This must appear after a pattern without SeqNumOffset.
	// The client derives sequece number by subtracting SeqNumOffset from the previous pattern's
	// sequence number. Sufficient CS capacity is necessary for Data to actually come from CS.
	SeqNumOffset int
}

// Server config.
type ServerConfig struct {
	Patterns []ServerPattern // traffic patterns
	Nack     bool            // whether to respond Nacks to unmatched Interests
}

// Server pattern definition.
type ServerPattern struct {
	Prefix  *ndn.Name     // name prefix
	Replies []ServerReply // reply settings
}

// Server reply definition.
type ServerReply struct {
	Weight int // weight of random choice, minimum is 1

	Suffix          *ndn.Name     // suffix to append to Interest name
	FreshnessPeriod time.Duration // FreshnessPeriod value
	PayloadLen      int           // Content payload length

	Nack ndn.NackReason // if not NackReason_None, reply with Nack instead of Data

	Timeout bool // if true, drop the Interest instead of sending Data
}
