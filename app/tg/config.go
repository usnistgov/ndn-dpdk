package tg

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/fileserver"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/multierr"
)

// Config describes traffic generator configuration.
type Config struct {
	Face       iface.LocatorWrapper `json:"face"`
	Producer   *tgproducer.Config   `json:"producer,omitempty"`
	FileServer *fileserver.Config   `json:"fileServer,omitempty"`
	Consumer   *tgconsumer.Config   `json:"consumer,omitempty"`
	Fetcher    *fetch.FetcherConfig `json:"fetcher,omitempty"`
}

// Validate applies defaults and validates the configuration.
func (cfg *Config) Validate() error {
	errs := []error{}
	hasProducer, hasConsumer := "", ""

	type validator interface {
		Validate() error
	}
	validate := func(field string, v validator, sameKind *string) {
		if reflect.ValueOf(v).IsNil() {
			return
		}
		if sameKind != nil {
			if *sameKind != "" {
				errs = append(errs, fmt.Errorf("%s and %s cannot coexist", *sameKind, field))
			}
			*sameKind = field
		}
		if e := v.Validate(); e != nil {
			errs = append(errs, fmt.Errorf("%s %w", field, e))
		}
	}

	if cfg.Face.Locator == nil {
		errs = append(errs, errors.New("face is missing"))
	} else {
		validate("face", &cfg.Face, nil)
	}
	validate("producer", cfg.Producer, &hasProducer)
	validate("fileServer", cfg.FileServer, &hasProducer)
	validate("consumer", cfg.Consumer, &hasConsumer)
	validate("fetcher", cfg.Fetcher, &hasConsumer)
	if hasProducer == "" && hasConsumer == "" {
		errs = append(errs, errors.New("at least one producer or consumer module should be enabled"))
	}

	return multierr.Combine(errs...)
}
