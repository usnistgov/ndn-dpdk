package ifacetest

/*
#include "../face.h"

static bool
FaceTableTestFace_Close(Face* face)
{
  return true;
}

static const FaceOps faceTableTestFaceOps = {
  .close = FaceTableTestFace_Close,
};

void
FaceTableTestFace_Init(Face* face, FaceId id)
{
	face->id = id;
  face->ops = &faceTableTestFaceOps;
}
*/
import "C"
import (
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type FaceTableTestFace struct {
	iface.Face
}

func newFaceTableTestFace(id iface.FaceId) (face FaceTableTestFace) {
	face.AllocCFace(C.sizeof_Face, dpdk.NUMA_SOCKET_ANY)
	C.FaceTableTestFace_Init((*C.Face)(face.GetPtr()), C.FaceId(id))
	return face
}
