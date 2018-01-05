package main

/*
#include "server.h"
*/
import "C"
import (
	"ndn-dpdk/iface"
)

type NdnpingServer struct {
	c *C.NdnpingServer
}

func NewNdnpingServer(face iface.Face) (server NdnpingServer) {
	server.c = new(C.NdnpingServer)
	server.c.face = (*C.Face)(face.GetPtr())
	return server
}

func (server NdnpingServer) Run() int {
	return int(C.NdnpingServer_run(server.c))
}
