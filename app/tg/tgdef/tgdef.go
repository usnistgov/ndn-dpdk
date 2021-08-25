// Package tgdef contains common definitions and helpers for traffic generator.
package tgdef

import (
	"io"

	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// Role names.
const (
	RoleConsumer = "CONSUMER"
	RoleProducer = "PRODUCER"
)

// Module represents a traffic generator module.
type Module interface {
	io.Closer

	Face() iface.Face

	Workers() (list []ealthread.ThreadWithRole)
	Launch()
	Stop() error
}

// Producer represents a producer module.
type Producer interface {
	Module
	ConnectRxQueues(demuxI *iface.InputDemux)
}

// Consumer represents a consumer module.
type Consumer interface {
	Module
	ConnectRxQueues(demuxD, demuxN *iface.InputDemux)
}
