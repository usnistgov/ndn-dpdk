package tg

import (
	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/multierr"
)

// Task contains consumer and producer on a face.
type Task struct {
	Face      iface.Face
	Producers []*tgproducer.Producer
	Consumer  *tgconsumer.Consumer
	Fetch     *fetch.Fetcher
}

func newTask(face iface.Face, cfg TaskConfig) (task *Task, e error) {
	socket := face.NumaSocket()
	task = &Task{
		Face: face,
	}

	if cfg.Producer != nil {
		nThreads := math.MaxInt(1, cfg.Producer.NThreads)
		for i := 0; i < nThreads; i++ {
			p, e := tgproducer.New(task.Face, i, cfg.Producer.RxQueue)
			if e != nil {
				return nil, e
			}
			if e = p.SetPatterns(cfg.Producer.Patterns); e != nil {
				return nil, e
			}
			p.SetLCore(ealthread.DefaultAllocator.Alloc(roleProducer, socket))
			task.Producers = append(task.Producers, p)
		}
	}

	if cfg.Consumer != nil {
		c, e := tgconsumer.New(task.Face, cfg.Consumer.RxQueue)
		if e != nil {
			return nil, e
		}
		if e = c.SetPatterns(cfg.Consumer.Patterns); e != nil {
			return nil, e
		}
		if e = c.SetInterval(cfg.Consumer.Interval.Duration()); e != nil {
			return nil, e
		}
		c.SetLCores(ealthread.DefaultAllocator.Alloc(roleConsumer, socket), ealthread.DefaultAllocator.Alloc(roleConsumer, socket))
		task.Consumer = c
	} else if cfg.Fetch != nil {
		if task.Fetch, e = fetch.New(task.Face, *cfg.Fetch); e != nil {
			return nil, e
		}
		for i, last := 0, task.Fetch.CountThreads(); i < last; i++ {
			task.Fetch.Thread(i).SetLCore(ealthread.DefaultAllocator.Alloc(roleConsumer, socket))
		}
	}

	return task, nil
}

func (task *Task) configureDemux(demuxI, demuxD, demuxN *iface.InputDemux) {
	if nProducers := len(task.Producers); nProducers > 0 {
		demuxI.InitRoundrobin(nProducers)
		for i, producer := range task.Producers {
			demuxI.SetDest(i, producer.RxQueue())
		}
	}

	if task.Consumer != nil {
		demuxD.InitFirst()
		demuxN.InitFirst()
		q := task.Consumer.RxQueue()
		demuxD.SetDest(0, q)
		demuxN.SetDest(0, q)
	} else if task.Fetch != nil {
		demuxD.InitToken()
		demuxN.InitToken()
		for i, last := 0, task.Fetch.CountProcs(); i < last; i++ {
			q := task.Fetch.RxQueue(i)
			demuxD.SetDest(i, q)
			demuxN.SetDest(i, q)
		}
	}
}

func (task *Task) launch() {
	for _, p := range task.Producers {
		p.Launch()
	}
	if task.Consumer != nil {
		task.Consumer.Launch()
	}
}

func (task *Task) close() error {
	errs := []error{}
	for _, p := range task.Producers {
		errs = append(errs, p.Close())
	}
	if task.Consumer != nil {
		errs = append(errs, task.Consumer.Close())
	}
	if task.Fetch != nil {
		errs = append(errs, task.Fetch.Close())
	}
	errs = append(errs, task.Face.Close())
	return multierr.Combine(errs...)
}
