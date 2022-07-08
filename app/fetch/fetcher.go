// Package fetch simulates bulk file transfer traffic patterns.
package fetch

/*
#include "../../csrc/fetch/fetcher.h"
*/
import "C"
import (
	"errors"
	"math"

	"github.com/usnistgov/ndn-dpdk/app/tg/tgdef"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/zyedidia/generic"
	"go.uber.org/multierr"
	"golang.org/x/exp/maps"
)

var logger = logging.New("fetch")

// Config contains Fetcher configuration.
type Config struct {
	TaskSlotConfig

	// NThreads is the number of worker threads.
	// Each worker thread can serve multiple fetch tasks.
	NThreads int `json:"nThreads,omitempty"`

	// NTasks is the number of task slots.
	// Each task retrieves one segmented object and has independent congestion control.
	NTasks int `json:"nTasks,omitempty"`
}

// Validate applies defaults and validates the configuration.
func (cfg *Config) Validate() error {
	cfg.TaskSlotConfig.applyDefaults()
	cfg.NThreads = generic.Clamp(cfg.NThreads, 1, math.MaxInt8)
	cfg.NTasks = generic.Clamp(cfg.NTasks, 1, iface.MaxInputDemuxDest)
	return nil
}

// Fetcher controls worker threads and task slots on a face.
type Fetcher struct {
	workers   []*worker
	taskSlots []*taskSlot
}

var _ tgdef.Consumer = &Fetcher{}

// Face returns the face.
func (fetcher *Fetcher) Face() iface.Face {
	return fetcher.workers[0].Face()
}

// ConnectRxQueues connects Data InputDemux to RxQueues.
// Nack InputDemux is set to drop packets because fetcher does not support Nacks.
func (fetcher *Fetcher) ConnectRxQueues(demuxD, demuxN *iface.InputDemux) {
	demuxD.InitToken(0)
	for i, ts := range fetcher.taskSlots {
		demuxD.SetDest(i, ts.RxQueueD())
	}
	demuxN.InitDrop()
}

// Workers returns worker threads.
func (fetcher *Fetcher) Workers() []ealthread.ThreadWithRole {
	return tgdef.GatherWorkers(fetcher.workers)
}

// Tasks returns running fetch tasks.
func (fetcher *Fetcher) Tasks() (list []*TaskContext) {
	taskContextLock.RLock()
	defer taskContextLock.RUnlock()
	list = []*TaskContext{}
	for _, task := range taskContextByID {
		if task.fetcher == fetcher {
			list = append(list, task)
		}
	}
	return
}

// Fetch starts a fetch task.
func (fetcher *Fetcher) Fetch(d TaskDef) (task *TaskContext, e error) {
	eal.CallMain(func() {
		task = &TaskContext{
			d:        d,
			fetcher:  fetcher,
			w:        fetcher.workers[0],
			stopping: make(chan struct{}),
		}

		for _, ts := range fetcher.taskSlots {
			if ts.worker == -1 {
				task.ts = ts
				break
			}
		}
		if task.ts == nil {
			task, e = nil, errors.New("too many running tasks")
			return
		}
		if e = task.ts.Init(d); e != nil {
			task = nil
			return
		}

		for _, w := range fetcher.workers {
			if task.w.nTasks > w.nTasks {
				task.w = w
			}
		}

		task.w.AddTask(eal.MainReadSide, task.ts)
		e = nil
		taskContextLock.Lock()
		defer taskContextLock.Unlock()
		lastTaskContextID++
		task.id = lastTaskContextID
		taskContextByID[task.id] = task
	})
	return
}

// Launch launches all worker threads.
func (fetcher *Fetcher) Launch() {
	tgdef.LaunchWorkers(fetcher.workers)
}

// Stop stops all worker threads.
func (fetcher *Fetcher) Stop() error {
	return tgdef.StopWorkers(fetcher.workers)
}

// Reset aborts all tasks and stops all worker threads.
func (fetcher *Fetcher) Reset() {
	fetcher.Stop()
	for _, w := range fetcher.workers {
		w.ClearTasks()
	}
	for _, ts := range fetcher.taskSlots {
		ts.closeFd()
		ts.worker = -1
	}
	maps.DeleteFunc(taskContextByID, func(id int, task *TaskContext) bool { return task.fetcher == fetcher })
}

// Close deallocates data structures.
func (fetcher *Fetcher) Close() error {
	errs := []error{
		fetcher.Stop(),
	}
	for _, ts := range fetcher.taskSlots {
		errs = append(errs,
			ts.RxQueueD().Close(),
			ts.Logic().Close(),
		)
		eal.Free(ts)
	}
	for _, w := range fetcher.workers {
		eal.Free(w.c)
	}
	return multierr.Combine(errs...)
}

// New creates a Fetcher.
func New(face iface.Face, cfg Config) (*Fetcher, error) {
	cfg.applyDefaults()

	fetcher := &Fetcher{
		workers:   make([]*worker, cfg.NThreads),
		taskSlots: make([]*taskSlot, cfg.NTasks),
	}
	for i := range fetcher.workers {
		fetcher.workers[i] = newWorker(face, i)
	}

	socket := face.NumaSocket()
	for i := range fetcher.taskSlots {
		fetcher.taskSlots[i] = newTaskSlot(i, cfg.TaskSlotConfig, socket)
	}

	return fetcher, nil
}
