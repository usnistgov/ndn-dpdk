package main

import (
	"fmt"
	stdlog "log"
	"os"
	"time"

	"ndn-dpdk/app/ndnping"
	"ndn-dpdk/dpdk"
)

func main() {
	pc, e := parseCommand(dpdk.MustInitEal(os.Args)[1:])
	if e != nil {
		log.WithError(e).Fatal("command line error")
	}

	pc.initCfg.Apply()

	app, e := ndnping.New(pc.tasks)
	if e != nil {
		log.WithError(e).Fatal("ndnping.NewApp error")
	}

	app.Launch()

	if pc.counterInterval > 0 {
		go printPeriodicCounters(app, pc.counterInterval)
	}

	if pc.wantThroughputBenchmark() {
		tb := NewThroughputBenchmark(app.Tasks[0].Client, pc.throughputBenchmark)
		if ok, msi, cnt := tb.Run(); ok {
			fmt.Println(msi.Nanoseconds())
			fmt.Println(cnt)
			os.Exit(0)
		} else {
			os.Exit(3)
		}
	}

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
