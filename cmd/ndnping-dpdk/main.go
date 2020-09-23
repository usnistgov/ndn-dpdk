package main

import (
	stdlog "log"
	"os"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/ping"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealinit"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/mgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/facemgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/pingmgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/versionmgmt"
)

func main() {
	gqlserver.Start()

	pc, e := parseCommand(ealinit.Init(os.Args)[1:])
	if e != nil {
		log.WithError(e).Fatal("command line error")
	}

	pc.initCfg.Mempool.Apply()
	ealthread.DefaultAllocator.Config = pc.initCfg.LCoreAlloc

	app, e := ping.New(pc.tasks)
	if e != nil {
		log.WithError(e).Fatal("ping.NewApp error")
	}

	app.Launch()

	if pc.counterInterval > 0 {
		go printPeriodicCounters(app, pc.counterInterval)
	}

	mgmt.Register(versionmgmt.VersionMgmt{})
	mgmt.Register(facemgmt.FaceMgmt{})
	mgmt.Register(pingmgmt.PingClientMgmt{App: app})
	mgmt.Register(pingmgmt.FetchMgmt{App: app})
	mgmt.Start()

	select {}
}

func printPeriodicCounters(app *ping.App, counterInterval time.Duration) {
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
