#include "mock-face.h"
#include "_cgo_export.h"

#include "../../core/logger.h"

INIT_ZF_LOG(MockFace);

static int
MockFace_GetNumaSocket(Face* faceBase)
{
  return SOCKET_ID_ANY;
}

static const FaceOps MockFaceOps = {
  .close = go_MockFace_Close,
  .getNumaSocket = MockFace_GetNumaSocket,
};

void
MockFace_Init(MockFace* face, FaceId id, FaceMempools* mempools)
{
  ZF_LOGI("%p Init(id=%" PRI_FaceId ")", face, id);

  face->base.id = id;
  face->base.txBurstOp = go_MockFace_TxBurst;
  face->base.ops = &MockFaceOps;

  FaceImpl_Init(&face->base, 0, 0, mempools);
}

static FaceRxBurst* theRxBurst = NULL;

void
MockFace_Rx(MockFace* face, void* cb, void* cbarg, Packet* npkt)
{
  if (unlikely(theRxBurst == NULL)) {
    theRxBurst = FaceRxBurst_New(1);
  }
  FaceRxBurst_GetScratch(theRxBurst)[0] = Packet_ToMbuf(npkt);
  FaceImpl_RxBurst(&face->base, theRxBurst, 1, (Face_RxCb)cb, cbarg);
}
