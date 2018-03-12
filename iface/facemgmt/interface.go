package facemgmt

import (
	"ndn-dpdk/iface"
)

type IFaceTable interface {
	ListFaces() []iface.Face
	GetFace(id iface.FaceId) iface.Face
}
