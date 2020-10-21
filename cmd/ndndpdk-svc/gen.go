package main

import (
	"errors"
	stdlog "log"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/ping"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealconfig"
	"github.com/usnistgov/ndn-dpdk/mgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/pingmgmt"
)

var (
	errGenNoTasks = errors.New("tasks missing")
)

type genArgs struct {
	CommonArgs
	Tasks           []ping.TaskConfig       `json:"tasks"`
	CounterInterval nnduration.Milliseconds `json:"counterInterval,omitempty"`
}

func (a genArgs) Activate() error {
	if len(a.Tasks) == 0 {
		return errGenNoTasks
	}

	var req ealconfig.Request
	req.MinLCores = 1 // main
	for _, task := range a.Tasks {
		req.MinLCores += task.EstimateLCores()
	}
	if e := a.CommonArgs.apply(req); e != nil {
		return e
	}

	app, e := ping.New(a.Tasks)
	if e != nil {
		return e
	}
	app.Launch()
	mgmt.Register(pingmgmt.PingClientMgmt{App: app})
	mgmt.Register(pingmgmt.FetchMgmt{App: app})
	mgmt.Start()

	go printPingCounters(app, a.CounterInterval.DurationOr(1000))
	return nil
}

func printPingCounters(app *ping.App, counterInterval time.Duration) {
	for range time.Tick(counterInterval) {
		for _, task := range app.Tasks {
			face := task.Face
			stdlog.Printf("face(%d): %v %v", face.ID(), face.ReadCounters(), face.ReadExCounters())
			for i, server := range task.Server {
				stdlog.Printf("  server[%d]: %v", i, server.ReadCounters())
			}
			if client := task.Client; client != nil {
				stdlog.Printf("  client: %v", client.ReadCounters())
			} else if fetcher := task.Fetch; fetcher != nil {
				for i, last := 0, fetcher.CountProcs(); i < last; i++ {
					cnt := fetcher.Logic(i).ReadCounters()
					stdlog.Printf("  fetch[%d]: %v", i, cnt)
				}
			}
		}
	}
}
