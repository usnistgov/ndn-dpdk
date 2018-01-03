#include "socket-face.h"
#include "_cgo_export.h"

static const FaceOps socketFaceOps = {
  .close = go_SocketFace_Close,
};

void
SocketFace_Init(SocketFace* face, uint16_t id, struct rte_mempool* indirectMp,
                struct rte_mempool* headerMp)
{
  face->base.id = id;
  face->base.rxBurstOp = go_SocketFace_RxBurst;
  face->base.txBurstOp = go_SocketFace_TxBurst;
  face->base.ops = &socketFaceOps;

  FaceImpl_Init(&face->base, 0, 0, indirectMp, headerMp);
}
