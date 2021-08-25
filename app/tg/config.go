package tg

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// Error conditions.
var (
	ErrNoFace          = errors.New("face locator is missing")
	ErrConsumerFetcher = errors.New("consumer and fetcher cannot coexist")
	ErrNoElement       = errors.New("at least one of producer, consumer, and fetcher should be specified")
)

// Config describes traffic generator configuration.
type Config struct {
	Face     iface.LocatorWrapper `json:"face"`
	Producer *tgproducer.Config   `json:"producer,omitempty"`
	Consumer *tgconsumer.Config   `json:"consumer,omitempty"`
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
