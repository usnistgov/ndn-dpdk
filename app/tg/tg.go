// Package tg controls traffic generator elements.
package tg

import (
	"errors"
	"fmt"
	"sync"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/multierr"
)

var logger = logging.New("tg")

var (
	mapFaceGen      = make(map[iface.ID]*TrafficGen)
	mapFaceGenMutex sync.RWMutex
)

func saveChooseRxlTxl() (undo func()) {
	oldChooseRxLoop, oldChooseTxLoop := iface.ChooseRxLoop, iface.ChooseTxLoop
	return func() { iface.ChooseRxLoop, iface.ChooseTxLoop = oldChooseRxLoop, oldChooseTxLoop }
}

// TrafficGen represents the traffic generator on a face.
type TrafficGen struct {
	face    iface.Face
	rxl     iface.RxLoop
	txl     iface.TxLoop
	workers []ealthread.ThreadWithRole

	producer *tgproducer.Producer
	consumer *tgconsumer.Consumer
	fetcher  *fetch.Fetcher
	exit     chan struct{}
}

// Face returns the face on which this traffic generator operates.
func (gen TrafficGen) Face() iface.Face {
	return gen.face
}

// Producer returns the producer element.
func (gen TrafficGen) Producer() *tgproducer.Producer {
	return gen.producer
}

// Consumer returns the fixed rate consumer element.
func (gen TrafficGen) Consumer() *tgconsumer.Consumer {
	return gen.consumer
}

// Fetcher returns the congestion aware fetcher element.
func (gen TrafficGen) Fetcher() *fetch.Fetcher {
	return gen.fetcher
}

// Workers implements tggql.withCommonFields interface.
func (gen TrafficGen) Workers() []ealthread.ThreadWithRole {
	return gen.workers
}

// Launch launches the traffic generator.
func (gen *TrafficGen) Launch() error {
	gen.txl.Launch()
	gen.configureDemux(gen.rxl.InterestDemux(), gen.rxl.DataDemux(), gen.rxl.NackDemux())
	gen.rxl.Launch()

	if gen.producer != nil {
		gen.producer.Launch()
	}
	if gen.consumer != nil {
		gen.consumer.Launch()
	}

	return nil
}

func (gen *TrafficGen) configureDemux(demuxI, demuxD, demuxN *iface.InputDemux) {
	demuxI.InitDrop()
	demuxD.InitDrop()
	demuxN.InitDrop()

	if gen.producer != nil {
		gen.producer.ConnectRxQueues(demuxI)
	}

	if gen.consumer != nil {
		demuxD.InitFirst()
		demuxN.InitFirst()
		q := gen.consumer.RxQueue()
		demuxD.SetDest(0, q)
		demuxN.SetDest(0, q)
	} else if gen.fetcher != nil {
		gen.fetcher.ConnectRxQueues(demuxD, demuxN)
	}
}

// Stop stops the traffic generator.
// It can be launched again.
func (gen *TrafficGen) Stop() error {
	errs := []error{}
	if gen.producer != nil {
		errs = append(errs, gen.producer.Stop())
	}
	if gen.consumer != nil {
		errs = append(errs, gen.consumer.Stop(0))
	}
	if gen.fetcher != nil {
		errs = append(errs, gen.fetcher.Stop())
	}
	return multierr.Combine(errs...)
}

// Close releases resources.
// It cannot be launched again.
func (gen *TrafficGen) Close() error {
	if e := gen.Stop(); e != nil {
		return e
	}
	close(gen.exit)

	if gen.face != nil {
		mapFaceGenMutex.Lock()
		delete(mapFaceGen, gen.face.ID())
		mapFaceGenMutex.Unlock()
	}

	errs := []error{}
	if gen.producer != nil {
		errs = append(errs, gen.producer.Close())
	}
	if gen.consumer != nil {
		errs = append(errs, gen.consumer.Close())
	}
	if gen.fetcher != nil {
		errs = append(errs, gen.fetcher.Close())
	}
	if gen.face != nil {
		errs = append(errs, gen.face.Close())
	}
	if gen.rxl != nil {
		errs = append(errs, gen.rxl.Close())
	}
	if gen.txl != nil {
		errs = append(errs, gen.txl.Close())
	}

	for _, w := range gen.workers {
		if lc := w.LCore(); lc.Valid() {
			ealthread.DefaultAllocator.Free(w.LCore())
		}
	}

	*gen = TrafficGen{}
	return multierr.Combine(errs...)
}

// New creates a traffic generator.
func New(cfg Config) (gen *TrafficGen, e error) {
	if e = cfg.Validate(); e != nil {
		return nil, e
	}

	gen = &TrafficGen{
		exit: make(chan struct{}),
	}
	success := false
	defer func(gen *TrafficGen) {
		if !success {
			gen.Close()
		}
	}(gen)

	defer saveChooseRxlTxl()()
	iface.ChooseRxLoop = func(rxg iface.RxGroup) iface.RxLoop {
		gen.rxl = iface.NewRxLoop(rxg.NumaSocket())
		return gen.rxl
	}
	iface.ChooseTxLoop = func(face iface.Face) iface.TxLoop {
		gen.txl = iface.NewTxLoop(face.NumaSocket())
		return gen.txl
	}

	if gen.face, e = cfg.Face.CreateFace(); e != nil {
		return nil, fmt.Errorf("error creating face %w", e)
	}
	if gen.rxl == nil {
		return nil, errors.New("face creation did not result in RxLoop creation")
	}
	if gen.txl == nil {
		return nil, errors.New("face creation did not result in TxLoop creation")
	}
	gen.workers = []ealthread.ThreadWithRole{gen.rxl, gen.txl}

	if cfg.Producer != nil {
		producer, e := tgproducer.New(gen.face, cfg.Producer.RxQueue, cfg.Producer.NThreads)
		if e != nil {
			return nil, fmt.Errorf("error creating producer %w", e)
		}
		if e = producer.SetPatterns(cfg.Producer.Patterns); e != nil {
			return nil, fmt.Errorf("error setting producer patterns %w", e)
		}
		gen.workers = append(gen.workers, producer.Workers()...)
		gen.producer = producer
	}

	if cfg.Consumer != nil {
		consumer, e := tgconsumer.New(gen.face, cfg.Consumer.RxQueue)
		if e != nil {
			return nil, fmt.Errorf("error creating consumer %w", e)
		}
		if e = consumer.SetPatterns(cfg.Consumer.Patterns); e != nil {
			return nil, fmt.Errorf("error setting consumer patterns %w", e)
		}
		if e = consumer.SetInterval(cfg.Consumer.Interval.Duration()); e != nil {
			return nil, fmt.Errorf("error setting consumer interval %w", e)
		}
		gen.workers = append(gen.workers, consumer.Workers()...)
		gen.consumer = consumer
	} else if cfg.Fetcher != nil {
		fetcher, e := fetch.New(gen.face, *cfg.Fetcher)
		if e != nil {
			return nil, fmt.Errorf("error creating fetcher %w", e)
		}
		gen.workers = append(gen.workers, fetcher.Workers()...)
		gen.fetcher = fetcher
	}

	if e := ealthread.AllocThread(gen.workers...); e != nil {
		return nil, fmt.Errorf("error allocating gen.workers %w", e)
	}

	success = true
	mapFaceGenMutex.Lock()
	defer mapFaceGenMutex.Unlock()
	mapFaceGen[gen.face.ID()] = gen
	return gen, nil
}

// Get retrieves traffic generator instance by face.
func Get(id iface.ID) *TrafficGen {
	mapFaceGenMutex.RLock()
	defer mapFaceGenMutex.RUnlock()
	return mapFaceGen[id]
}
