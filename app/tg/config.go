package tg

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// Error conditions.
var (
	ErrNoFace          = errors.New("face locator is missing")
	ErrConsumerFetcher = errors.New("consumer and fetcher cannot coexist")
	ErrNoElement       = errors.New("at least one of producer, consumer, and fetcher should be specified")
)

// ProducerConfig describes producer configuration.
type ProducerConfig struct {
	RxQueue  iface.PktQueueConfig `json:"rxQueue,omitempty"`
	Patterns []tgproducer.Pattern `json:"patterns"`
	NThreads int                  `json:"nThreads,omitempty"` // number of threads, minimum/default is 1
}

// ConsumerConfig describes consumer configuration.
type ConsumerConfig struct {
	RxQueue  iface.PktQueueConfig   `json:"rxQueue,omitempty"`
	Patterns []tgconsumer.Pattern   `json:"patterns"`
	Interval nnduration.Nanoseconds `json:"interval"`
}

// Config describes traffic generator configuration.
type Config struct {
	Face     iface.LocatorWrapper `json:"face"`
	Producer *ProducerConfig      `json:"producer,omitempty"`
	Consumer *ConsumerConfig      `json:"consumer,omitempty"`
	Fetcher  *fetch.FetcherConfig `json:"fetcher,omitempty"`
}

func (cfg Config) Validate() error {
	if cfg.Face.Locator == nil {
		return ErrNoFace
	}
	if e := cfg.Face.Validate(); e != nil {
		return e
	}

	if cfg.Consumer != nil && cfg.Fetcher != nil {
		return ErrConsumerFetcher
	}

	if cfg.Producer == nil && cfg.Consumer == nil && cfg.Fetcher == nil {
		return ErrNoElement
	}

	return nil
}
