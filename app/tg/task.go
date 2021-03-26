package tg

import (
	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go4.org/must"
)

// Task contains consumer and producer on a face.
type Task struct {
	Face     iface.Face
	Producer []*tgproducer.Producer
	Consumer *tgconsumer.Consumer
	Fetch    *fetch.Fetcher
}

func newTask(face iface.Face, cfg TaskConfig) (task *Task, e error) {
	socket := face.NumaSocket()
	task = &Task{
		Face: face,
	}

	if cfg.Producer != nil {
		nThreads := cfg.Producer.NThreads
		if nThreads <= 0 {
			nThreads = 1
		}
		for i := 0; i < nThreads; i++ {
			server, e := tgproducer.New(task.Face, i, cfg.Producer.Config)
			if e != nil {
				return nil, e
			}
			server.SetLCore(ealthread.DefaultAllocator.Alloc(roleProducer, socket))
			task.Producer = append(task.Producer, server)
		}
	}

	if cfg.Consumer != nil {
		if task.Consumer, e = tgconsumer.New(task.Face, *cfg.Consumer); e != nil {
			return nil, e
		}
		task.Consumer.SetLCores(ealthread.DefaultAllocator.Alloc(roleConsumer, socket), ealthread.DefaultAllocator.Alloc(roleConsumer, socket))
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
	if nServers := len(task.Producer); nServers > 0 {
		demuxI.InitRoundrobin(nServers)
		for i, server := range task.Producer {
			demuxI.SetDest(i, server.RxQueue())
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
	for _, server := range task.Producer {
		server.Launch()
	}
	if task.Consumer != nil {
		task.Consumer.Launch()
	}
}

func (task *Task) close() error {
	for _, server := range task.Producer {
		must.Close(server)
	}
	if task.Consumer != nil {
		must.Close(task.Consumer)
	}
	if task.Fetch != nil {
		must.Close(task.Fetch)
	}
	must.Close(task.Face)
	return nil
}
