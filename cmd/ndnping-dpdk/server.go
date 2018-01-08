package main

/*
#include "server.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/container/nameset"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type NdnpingServer struct {
	c *C.NdnpingServer
}

func NewNdnpingServer(face iface.Face) (server NdnpingServer) {
	server.c = new(C.NdnpingServer)
	server.c.face = (*C.Face)(face.GetPtr())
	return server
}

func (server NdnpingServer) SetNackNoRoute(enable bool) {
	server.c.wantNackNoRoute = C.bool(enable)
}

func (server NdnpingServer) AddPrefix(comps ndn.TlvBytes) {
	prefixSet := nameset.FromPtr(unsafe.Pointer(&server.c.prefixes))
	prefixSet.Insert(comps)
}

func (server NdnpingServer) Run() int {
	return int(C.NdnpingServer_run(server.c))
}
