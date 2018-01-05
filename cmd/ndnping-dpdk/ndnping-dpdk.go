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

	for _, server := range pc.servers {
		face, e := appinit.NewFaceFromUri(server.face)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewFaceFromUri(%s): %v", server.face, e)
		}

		server := NewNdnpingServer(*face)
		appinit.LaunchRequired(server.Run, face.GetNumaSocket())
	}

	select {}
}
