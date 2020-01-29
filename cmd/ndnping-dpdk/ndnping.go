package main

import (
	stdlog "log"
	"math"
	"os"
	"time"

	"ndn-dpdk/app/fetch"
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

	pc.initCfg.InitConfig.Apply()

	app, e := ping.New(pc.tasks, pc.initCfg.Ping)
	if e != nil {
		log.WithError(e).Fatal("ping.NewApp error")
	}

	app.Launch()

	if pc.counterInterval > 0 {
		go printPeriodicCounters(app, pc.counterInterval)
	}

	appinit.RegisterMgmt(versionmgmt.VersionMgmt{})
	appinit.RegisterMgmt(facemgmt.FaceMgmt{})
	appinit.RegisterMgmt(pingmgmt.PingClientMgmt{app})
	appinit.RegisterMgmt(pingmgmt.FetchMgmt{app})
	appinit.StartMgmt()

	select {}
}

func printPeriodicCounters(app *ping.App, counterInterval time.Duration) {
	prevFetchCnt := make(map[*fetch.Fetcher]fetch.Counters)
	for range time.Tick(counterInterval) {
		for _, task := range app.Tasks {
			face := task.Face
			stdlog.Printf("face(%d): %v %v", face.GetFaceId(), face.ReadCounters(), face.ReadExCounters())
			if fetcher := task.Fetch; fetcher != nil {
				cnt := fetcher.Logic.ReadCounters()
				goodput := math.NaN()
				if prev, ok := prevFetchCnt[fetcher]; ok {
					goodput = cnt.ComputeGoodput(prev)
				}
				prevFetchCnt[fetcher] = cnt
				stdlog.Printf("  fetch: %v %0.0fD/s", cnt, goodput)
			}
			if client := task.Client; client != nil {
				stdlog.Printf("  client: %v", client.ReadCounters())
			}
			if server := task.Server; server != nil {
				stdlog.Printf("  server: %v", server.ReadCounters())
			}
		}
	}
}
