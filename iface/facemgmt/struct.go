package facemgmt

import (
	"ndn-dpdk/iface"
)

type IdArg struct {
	Id iface.FaceId
}

type FaceInfo struct {
	Id       iface.FaceId
	Counters iface.Counters
}
