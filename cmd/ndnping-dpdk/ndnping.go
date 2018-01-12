package main

import (
	"log"
	"time"

	"ndn-dpdk/appinit"
)

func main() {
	appinit.InitEal()
	pc, e := parseCommand(appinit.Eal.Args[1:])
	if e != nil {
		appinit.Exitf(appinit.EXIT_BAD_CONFIG, "parseCommand: %v", e)
	}

	var clients []NdnpingClient
	for _, clientCfg := range pc.clients {
		face, e := appinit.NewFaceFromUri(clientCfg.face)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewFaceFromUri(%s): %v", clientCfg.face, e)
		}

		client, e := NewNdnpingClient(*face)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewNdnpingClient(%s): %v", clientCfg.face, e)
		}
		for _, pattern := range clientCfg.patterns {
			client.AddPattern(pattern.prefix, pattern.pct)
		}
		client.SetInterval(time.Millisecond)
		clients = append(clients, client)
	}

	var servers []NdnpingServer
	for _, serverCfg := range pc.servers {
		face, e := appinit.NewFaceFromUri(serverCfg.face)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewFaceFromUri(%s): %v", serverCfg.face, e)
		}

		server, e := NewNdnpingServer(*face)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewPingServer(%s): %v", serverCfg.face, e)
		}
		for _, prefix := range serverCfg.prefixes {
			server.AddPrefix(prefix)
		}
		servers = append(servers, server)
	}

	for _, server := range servers {
		appinit.LaunchRequired(server.Run, server.GetFace().GetNumaSocket())
	}
	time.Sleep(100 * time.Millisecond)
	for _, client := range clients {
		appinit.LaunchRequired(client.Run, client.GetFace().GetNumaSocket())
	}

	tick := time.Tick(pc.counterInterval)
	go func() {
		for {
			<-tick
			for _, client := range clients {
				face := client.GetFace()
				log.Printf("client(%d) %v", face.GetFaceId(), face.ReadCounters())
			}
			for _, server := range servers {
				face := server.GetFace()
				log.Printf("server(%d) %v", face.GetFaceId(), face.ReadCounters())
			}
		}
	}()

	select {}
}
