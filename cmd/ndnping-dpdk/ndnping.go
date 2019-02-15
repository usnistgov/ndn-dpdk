package main

import (
	stdlog "log"
	"os"
	"time"

	"ndn-dpdk/app/ndnping"
	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
)

func main() {
	pc, e := parseCommand(dpdk.MustInitEal(os.Args)[1:])
	if e != nil {
		log.WithError(e).Fatal("command line error")
	}

	pc.initcfg.Mempool.Apply()
	if e := appinit.EnableCreateFace(pc.initcfg.Face); e != nil {
		log.WithError(e).Fatal("appinit.EnableCreateFace error")
	}

	app, e := ndnping.NewApp(pc.tasks)
	if e != nil {
		log.WithError(e).Fatal("ndnping.NewApp error")
	}

	app.Launch()

	go func() {
		tick := time.Tick(pc.counterInterval)
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
	}()

	select {}
}
