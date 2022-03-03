package tg

import (
	"github.com/usnistgov/ndn-dpdk/app/fileserver"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
)

// Counters contains traffic generator counters.
type Counters struct {
	Producer   *tgproducer.Counters `json:"producer,omitempty"`
	FileServer *fileserver.Counters `json:"fileServer,omitempty"`
	Consumer   *tgconsumer.Counters `json:"consumer,omitempty"`
}

// Counters retrieves counters.
func (gen *TrafficGen) Counters() (cnt Counters) {
	if gen.producer != nil {
		c := gen.producer.Counters()
		cnt.Producer = &c
	}
	if gen.fileServer != nil {
		c := gen.fileServer.Counters()
		cnt.FileServer = &c
	}
	if gen.consumer != nil {
		c := gen.consumer.Counters()
		cnt.Consumer = &c
	}
	return cnt
}
