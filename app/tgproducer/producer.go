// Package tgproducer implements a traffic generator producer.
package tgproducer

import (
	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/multierr"
	"go4.org/must"
)

// Producer represents a traffic generator producer instance.
type Producer struct {
	workers  []*worker
	patterns []Pattern
}

// Patterns returns traffic patterns.
func (p Producer) Patterns() []Pattern {
	return p.patterns
}

// SetPatterns sets new traffic patterns.
// This can only be used when the threads are stopped.
func (p *Producer) SetPatterns(inputPatterns []Pattern) error {
	if len(inputPatterns) == 0 {
		return ErrNoPattern
	}
	if len(inputPatterns) > MaxPatterns {
		return ErrTooManyPatterns
	}
	patterns := []Pattern{}
	nDataGen := 0
	for _, pattern := range inputPatterns {
		sumWeight, nData := pattern.applyDefaults()
		if sumWeight > MaxSumWeight {
			return ErrTooManyWeights
		}
		nDataGen += nData
		if len(pattern.prefixV) > ndni.NameMaxLength {
			return ErrPrefixTooLong
		}
		patterns = append(patterns, pattern)
	}

	for _, w := range p.workers {
		if w.IsRunning() {
			return ealthread.ErrRunning
		}
	}

	payloadMp := ndni.PayloadMempool.Get(p.Face().NumaSocket())
	dataGenVec, e := payloadMp.Alloc(nDataGen * len(p.workers))
	if e != nil {
		return e
	}

	p.patterns = patterns
	for _, w := range p.workers {
		w.setPatterns(patterns, &dataGenVec)
	}
	return nil
}

// Face returns the associated face.
func (p Producer) Face() iface.Face {
	return p.workers[0].face()
}

// Workers returns worker threads.
func (p Producer) Workers() (list []ealthread.ThreadWithRole) {
	for _, w := range p.workers {
		list = append(list, w)
	}
	return list
}

// ConnectRxQueues connects Interest InputDemux to RxQueues.
func (p *Producer) ConnectRxQueues(demuxI *iface.InputDemux) {
	demuxI.InitRoundrobin(len(p.workers))
	for i, w := range p.workers {
		demuxI.SetDest(i, w.rxQueue())
	}
}

// Launch launches all workers.
func (p *Producer) Launch() {
	for _, w := range p.workers {
		w.Launch()
	}
}

// Stop stops all workers.
func (p *Producer) Stop() error {
	errs := []error{}
	for _, w := range p.workers {
		errs = append(errs, w.Stop())
	}
	return multierr.Combine(errs...)
}

// Close closes the producer.
func (p *Producer) Close() error {
	errs := []error{p.Stop()}
	for _, w := range p.workers {
		errs = append(errs, w.close())
	}
	p.workers = nil
	return multierr.Combine(errs...)
}

// New creates a Producer.
func New(face iface.Face, rxqCfg iface.PktQueueConfig, nWorkers int) (p *Producer, e error) {
	faceID := face.ID()
	socket := face.NumaSocket()
	rxqCfg.DisableCoDel = true
	nWorkers = math.MaxInt(1, nWorkers)

	p = &Producer{}
	for i := 0; i < nWorkers; i++ {
		w, e := newWorker(faceID, socket, rxqCfg)
		if e != nil {
			must.Close(p)
			return nil, e
		}
		p.workers = append(p.workers, w)
	}
	return p, nil
}
