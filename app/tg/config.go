package tg

import (
	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// TaskConfig describes traffic generator task on a face.
type TaskConfig struct {
	Face     iface.LocatorWrapper `json:"face"`
	Producer *ProducerConfig      `json:"producer,omitempty"`
	Consumer *tgconsumer.Config   `json:"consumer,omitempty"`
	Fetch    *fetch.FetcherConfig `json:"fetch,omitempty"`
}

// EstimateLCores estimates how many LCores are required to activate this task.
func (cfg TaskConfig) EstimateLCores() (n int) {
	n = 2 // RX + TX

	if cfg.Producer != nil {
		n += math.MaxInt(1, cfg.Producer.NThreads) // SVR
	}

	if cfg.Consumer != nil {
		n += 2 // CLIR + CLIT
	} else if cfg.Fetch != nil {
		n++ // CLIR
	}

	return n
}

// ProducerConfig describes traffic generator server task.
type ProducerConfig struct {
	tgproducer.Config
	NThreads int `json:"nThreads,omitempty"` // number of threads, minimum/default is 1
}
