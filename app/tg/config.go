package tg

import (
	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// TaskConfig describes traffic generator task on a face.
type TaskConfig struct {
	Face     iface.LocatorWrapper `json:"face"`
	Producer *struct {
		RxQueue  iface.PktQueueConfig `json:"rxQueue,omitempty"`
		Patterns []tgproducer.Pattern `json:"patterns"`
		NThreads int                  `json:"nThreads,omitempty"` // number of threads, minimum/default is 1
	} `json:"producer,omitempty"`
	Consumer *struct {
		RxQueue  iface.PktQueueConfig   `json:"rxQueue,omitempty"`
		Patterns []tgconsumer.Pattern   `json:"patterns"`
		Interval nnduration.Nanoseconds `json:"interval"`
	} `json:"consumer,omitempty"`
	Fetch *fetch.FetcherConfig `json:"fetch,omitempty"`
}

// EstimateLCores estimates how many LCores are required to activate this task.
func (cfg TaskConfig) EstimateLCores() (n int) {
	n = 2 // RX + TX

	if cfg.Producer != nil {
		n += math.MaxInt(1, cfg.Producer.NThreads)
	}

	if cfg.Consumer != nil {
		n += 2
	} else if cfg.Fetch != nil {
		n++
	}

	return n
}
