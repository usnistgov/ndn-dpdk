package main

import (
	"ndn-dpdk/appinit"
)

func main() {
	appinit.InitEal()
	pc, e := parseCommand(appinit.Eal.Args[1:])
	if e != nil {
		appinit.Exitf(appinit.EXIT_BAD_CONFIG, "parseCommand: %v", e)
	}

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
		appinit.LaunchRequired(client.Run, face.GetNumaSocket())
	}

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
		appinit.LaunchRequired(server.Run, face.GetNumaSocket())
	}

	select {}
}
