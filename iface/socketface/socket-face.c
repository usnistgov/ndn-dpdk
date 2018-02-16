#include "socket-face.h"
#include "_cgo_export.h"

static int
SocketFace_GetNumaSocket(Face* faceBase)
{
  return SOCKET_ID_ANY;
}

static const FaceOps socketFaceOps = {
  .close = go_SocketFace_Close,
  .getNumaSocket = SocketFace_GetNumaSocket,
};

void
SocketFace_Init(SocketFace* face, FaceId id, FaceMempools* mempools)
{
  face->base.id = id;
  face->base.rxBurstOp = go_SocketFace_RxBurst;
  face->base.txBurstOp = go_SocketFace_TxBurst;
  face->base.ops = &socketFaceOps;

  FaceImpl_Init(&face->base, 0, 0, mempools);
}
