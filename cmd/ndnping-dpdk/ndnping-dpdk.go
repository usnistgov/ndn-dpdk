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

	for _, serverCfg := range pc.servers {
		face, e := appinit.NewFaceFromUri(serverCfg.face)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewFaceFromUri(%s): %v", serverCfg.face, e)
		}

		server := NewNdnpingServer(*face)
		for _, prefix := range serverCfg.prefixes {
			server.AddPrefix(prefix)
		}
		appinit.LaunchRequired(server.Run, face.GetNumaSocket())
	}

	select {}
}
