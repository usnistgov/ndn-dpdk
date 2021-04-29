package main

import (
	stdlog "log"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/tg"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
)

type genArgs struct {
	CommonArgs
	Tasks           []tg.TaskConfig         `json:"tasks"`
	CounterInterval nnduration.Milliseconds `json:"counterInterval,omitempty"`
}

func (a genArgs) Activate() error {
	initXDPProgram()

	var req ealconfig.Request
	req.MinLCores = 1 // main
	for _, task := range a.Tasks {
		req.MinLCores += task.EstimateLCores()
	}
	if e := a.CommonArgs.apply(req); e != nil {
		return e
	}

	gen, e := tg.New(a.Tasks)
	if e != nil {
		return e
	}
	tg.GqlTrafficGen = gen

	gen.Launch()
	go printPingCounters(gen, a.CounterInterval.DurationOr(1000))
	return nil
}

func printPingCounters(gen *tg.TrafficGen, counterInterval time.Duration) {
	for range time.Tick(counterInterval) {
		for _, task := range gen.Tasks {
			face := task.Face
			stdlog.Printf("face(%d): %v %v", face.ID(), face.ReadCounters(), face.ReadExCounters())
			for i, producer := range task.Producers {
				stdlog.Printf("  producer[%d]: %v", i, producer.ReadCounters())
			}
			if consumer := task.Consumer; consumer != nil {
				stdlog.Printf("  consumer: %v", consumer.ReadCounters())
			} else if fetcher := task.Fetch; fetcher != nil {
				for i, last := 0, fetcher.CountProcs(); i < last; i++ {
					cnt := fetcher.Logic(i).ReadCounters()
					stdlog.Printf("  fetch[%d]: %v", i, cnt)
				}
			}
		}
	}
}
