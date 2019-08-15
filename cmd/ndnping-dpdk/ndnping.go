package main

import (
	stdlog "log"
	"os"
	"time"

	"ndn-dpdk/app/ndnping"
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

	app, e := ndnping.New(pc.tasks, pc.initCfg.Ping)
	if e != nil {
		log.WithError(e).Fatal("ndnping.NewApp error")
	}

	app.Launch()

	if pc.counterInterval > 0 {
		go printPeriodicCounters(app, pc.counterInterval)
	}

	appinit.RegisterMgmt(versionmgmt.VersionMgmt{})
	appinit.RegisterMgmt(facemgmt.FaceMgmt{})
	appinit.RegisterMgmt(pingmgmt.PingClientMgmt{app})
	appinit.StartMgmt()

	select {}
}

func printPeriodicCounters(app *ndnping.App, counterInterval time.Duration) {
	tick := time.Tick(counterInterval)
	for {
		<-tick
		for _, task := range app.Tasks {
			stdlog.Printf("face(%d): %v %v", task.Face.GetFaceId(),
				task.Face.ReadCounters(), task.Face.ReadExCounters())
			if task.Client != nil {
				stdlog.Printf("  client: %v", task.Client.ReadCounters())
			}
			if task.Server != nil {
				stdlog.Printf("  server: %v", task.Server.ReadCounters())
			}
		}
	}
}
