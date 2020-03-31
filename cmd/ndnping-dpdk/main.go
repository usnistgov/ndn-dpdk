package main

import (
	stdlog "log"
	"os"
	"time"

	"ndn-dpdk/app/ping"
	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/mgmt/facemgmt"
	"ndn-dpdk/mgmt/pingmgmt"
	"ndn-dpdk/mgmt/versionmgmt"
)

func main() {
	pc, e := parseCommand(dpdk.MustInitEal(os.Args)[1:])
	if e != nil {
		log.WithError(e).Fatal("command line error")
	}

	pc.initCfg.Apply()

	app, e := ping.New(pc.tasks)
	if e != nil {
		log.WithError(e).Fatal("ping.NewApp error")
	}

	app.Launch()

	if pc.counterInterval > 0 {
		go printPeriodicCounters(app, pc.counterInterval)
	}

	appinit.RegisterMgmt(versionmgmt.VersionMgmt{})
	appinit.RegisterMgmt(facemgmt.FaceMgmt{})
	appinit.RegisterMgmt(facemgmt.EthFaceMgmt{})
	appinit.RegisterMgmt(pingmgmt.PingClientMgmt{app})
	appinit.RegisterMgmt(pingmgmt.FetchMgmt{app})
	appinit.StartMgmt()

	select {}
}

func printPeriodicCounters(app *ping.App, counterInterval time.Duration) {
	for range time.Tick(counterInterval) {
		for _, task := range app.Tasks {
			face := task.Face
			stdlog.Printf("face(%d): %v %v", face.GetFaceId(), face.ReadCounters(), face.ReadExCounters())
			if server := task.Server; server != nil {
				stdlog.Printf("  server: %v", server.ReadCounters())
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
