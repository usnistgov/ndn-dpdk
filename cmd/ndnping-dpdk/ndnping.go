package main

import (
	stdlog "log"
	"time"

	"ndn-dpdk/app/ndnping"
	"ndn-dpdk/appinit"
)

func main() {
	appinit.InitEal()
	pc, e := parseCommand(appinit.Eal.Args[1:])
	if e != nil {
		log.WithError(e).Fatal("command line error")
	}
	pc.initcfg.Mempool.Apply()

	var clients []ndnping.Client
	for _, clientCfg := range pc.clients {
		logEntry := log.WithField("face", clientCfg.face)
		face, e := appinit.NewFaceFromUri(clientCfg.face, nil)
		if e != nil {
			logEntry.WithError(e).Fatal("client face creation error")
		}

		client, e := ndnping.NewClient(face)
		if e != nil {
			logEntry.WithError(e).Fatal("client initialization error")
		}
		client.SetInterval(clientCfg.interval)
		for _, pattern := range clientCfg.patterns {
			client.AddPattern(pattern.prefix, pattern.pct)
		}
		client.EnableRtt(8, 16)

		clients = append(clients, client)
	}

	var servers []ndnping.Server
	for _, serverCfg := range pc.servers {
		logEntry := log.WithField("face", serverCfg.face)
		face, e := appinit.NewFaceFromUri(serverCfg.face, nil)
		if e != nil {
			logEntry.WithError(e).Fatal("server face creation error")
		}

		server, e := ndnping.NewServer(face)
		if e != nil {
			logEntry.WithError(e).Fatal("server initialization error")
		}
		for _, prefix := range serverCfg.prefixes {
			server.AddPattern(prefix)
		}
		server.SetNackNoRoute(pc.serverNack)
		server.SetPayloadLen(pc.payloadLen)
		servers = append(servers, server)
	}

	for i, server := range servers {
		lc := appinit.MustLaunch(server.Run, server.GetFace().GetNumaSocket())
		log.WithFields(makeLogFields("server", i, "lcore", lc, "socket", lc.GetNumaSocket())).Info("server launch")
	}
	time.Sleep(100 * time.Millisecond)
	for i, client := range clients {
		lc1 := appinit.MustLaunch(client.RunRx, client.GetFace().GetNumaSocket())
		lc2 := appinit.MustLaunch(client.RunTx, lc1.GetNumaSocket())
		log.Printf("client(%d) lcore %d, %d socket %d", i, lc1, lc2, lc1.GetNumaSocket())
		log.WithFields(makeLogFields("client", i, "lcore-rx", lc1, "lcore-tx", lc2, "socket", lc1.GetNumaSocket())).Info("client launch")
	}

	tick := time.Tick(pc.counterInterval)
	go func() {
		for {
			<-tick
			for _, client := range clients {
				face := client.GetFace()
				stdlog.Printf("client(%d) %v; %v; %v", face.GetFaceId(),
					client.ReadCounters(), face.ReadCounters(), face.ReadExCounters())
			}
			for _, server := range servers {
				face := server.GetFace()
				stdlog.Printf("server(%d) %v; %v; %v", face.GetFaceId(),
					server.ReadCounters(), face.ReadCounters(), face.ReadExCounters())
			}
		}
	}()

	select {}
}
