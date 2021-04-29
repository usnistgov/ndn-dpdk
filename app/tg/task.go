package tg

import (
	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/multierr"
)

// Task contains consumer and producer on a face.
type Task struct {
	Face     iface.Face
	Producer *tgproducer.Producer
	Consumer *tgconsumer.Consumer
	Fetch    *fetch.Fetcher
}

func newTask(face iface.Face, cfg TaskConfig) (task *Task, e error) {
	task = &Task{Face: face}

	if cfg.Producer != nil {
		p, e := tgproducer.New(task.Face, cfg.Producer.RxQueue, cfg.Producer.NThreads)
		if e != nil {
			return nil, e
		}
		if e = p.SetPatterns(cfg.Producer.Patterns); e != nil {
			return nil, e
		}
		if e = ealthread.AllocThread(p.Workers()...); e != nil {
			return nil, e
		}
		task.Producer = p
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
		if e = ealthread.AllocThread(c.Workers()...); e != nil {
			return nil, e
		}
		task.Consumer = c
	} else if cfg.Fetch != nil {
		if task.Fetch, e = fetch.New(task.Face, *cfg.Fetch); e != nil {
			return nil, e
		}
		if e = ealthread.AllocThread(task.Fetch.Workers()...); e != nil {
			return nil, e
		}
	}

	return task, nil
}

func (task *Task) configureDemux(demuxI, demuxD, demuxN *iface.InputDemux) {
	if task.Producer != nil {
		demuxI.InitRoundrobin(len(task.Producer.Workers()))
		task.Producer.ConnectRxQueues(demuxI, 0)
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
		task.Fetch.ConnectRxQueues(demuxD, demuxN, 0)
	}
}

func (task *Task) launch() {
	if task.Producer != nil {
		task.Producer.Launch()
	}
	if task.Consumer != nil {
		task.Consumer.Launch()
	}
}

func (task *Task) close() error {
	errs := []error{}
	if task.Producer != nil {
		errs = append(errs, task.Producer.Close())
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
