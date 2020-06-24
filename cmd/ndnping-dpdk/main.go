package main

import (
	stdlog "log"
	"os"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/ping"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/mgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/facemgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/pingmgmt"
	"github.com/usnistgov/ndn-dpdk/mgmt/versionmgmt"
)

func main() {
	pc, e := parseCommand(eal.InitEal(os.Args)[1:])
	if e != nil {
		log.WithError(e).Fatal("command line error")
	}

	pc.initCfg.Mempool.Apply()
	eal.LCoreAlloc.Config = pc.initCfg.LCoreAlloc
	pc.initCfg.Face.Apply()

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
	mgmt.Register(facemgmt.EthFaceMgmt{})
	mgmt.Register(pingmgmt.PingClientMgmt{app})
	mgmt.Register(pingmgmt.FetchMgmt{app})
	mgmt.Start()

	select {}
}

func printPeriodicCounters(app *ping.App, counterInterval time.Duration) {
	for range time.Tick(counterInterval) {
		for _, task := range app.Tasks {
			face := task.Face
			stdlog.Printf("face(%d): %v %v", face.GetFaceId(), face.ReadCounters(), face.ReadExCounters())
			for i, server := range task.Server {
				stdlog.Printf("  server[%d]: %v", i, server.ReadCounters())
			}
			if client := task.Client; client != nil {
				stdlog.Printf("  client: %v", client.ReadCounters())
			} else if fetcher := task.Fetch; fetcher != nil {
				for i, last := 0, fetcher.CountProcs(); i < last; i++ {
					cnt := fetcher.GetLogic(i).ReadCounters()
					stdlog.Printf("  fetch[%d]: %v", i, cnt)
				}
			}
		}
	}
}
