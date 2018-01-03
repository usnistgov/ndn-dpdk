#include "socket-face.h"
#include "_cgo_export.h"

static const FaceOps socketFaceOps = {
  .close = go_SocketFace_Close,
};

void
SocketFace_Init(SocketFace* face, uint16_t id)
{
  face->base.id = id;
  face->base.rxBurstOp = go_SocketFace_RxBurst;
  face->base.txBurstOp = go_SocketFace_TxBurst;
  face->base.ops = &socketFaceOps;
}
