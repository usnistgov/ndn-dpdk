// Package tgproducer implements a traffic generator producer.
package tgproducer

import (
	"errors"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/app/tg/tgdef"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go4.org/must"
)

// Producer represents a traffic generator producer instance.
type Producer struct {
	cfg     Config
	workers []*worker
}

var _ tgdef.Producer = &Producer{}

// Patterns returns traffic patterns.
func (p Producer) Patterns() []Pattern {
	return p.cfg.Patterns
}

func (p *Producer) initPatterns() error {
	payloadMp := ndni.PayloadMempool.Get(p.Face().NumaSocket())
	dataGenVec, e := payloadMp.Alloc(p.cfg.nDataGen * len(p.workers))
	if e != nil {
		return e
	}

	for _, w := range p.workers {
		w.setPatterns(p.cfg.Patterns, dataGenVec.Take)
	}
	return nil
}

// Face returns the associated face.
func (p Producer) Face() iface.Face {
	return p.workers[0].face()
}

// ConnectRxQueues connects Interest InputDemux to RxQueues.
func (p *Producer) ConnectRxQueues(demuxI *iface.InputDemux) {
	demuxI.InitRoundrobin(len(p.workers))
	for i, w := range p.workers {
		demuxI.SetDest(i, w.rxQueue())
	}
}

// Workers returns worker threads.
func (p Producer) Workers() []ealthread.ThreadWithRole {
	return tgdef.GatherWorkers(p.workers)
}

// Launch launches all workers.
func (p *Producer) Launch() {
	tgdef.LaunchWorkers(p.workers)
}

// Stop stops all workers.
func (p *Producer) Stop() error {
	return tgdef.StopWorkers(p.workers)
}

// Close closes the producer.
func (p *Producer) Close() error {
	errs := []error{p.Stop()}
	for _, w := range p.workers {
		errs = append(errs, w.close())
	}
	p.workers = nil
	return errors.Join(errs...)
}

// New creates a Producer.
func New(face iface.Face, cfg Config) (p *Producer, e error) {
	if e := cfg.Validate(); e != nil {
		return nil, e
	}

	faceID := face.ID()
	socket := face.NumaSocket()

	p = &Producer{
		cfg: cfg,
	}
	for range cfg.NThreads {
		w, e := newWorker(faceID, socket, cfg.RxQueue)
		if e != nil {
			must.Close(p)
			return nil, e
		}
		p.workers = append(p.workers, w)
	}

	if e := p.initPatterns(); e != nil {
		must.Close(p)
		return nil, fmt.Errorf("error setting patterns %w", e)
	}
	return p, nil
}
