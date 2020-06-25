#include "face.h"

Face gFaces_[FACEID_MAX + 1];

void
FaceImpl_RxBurst(FaceRxBurst* burst, uint16_t nFrames, int rxThread, Face_RxCb cb, void* cbarg)
{
  FaceRxBurst_Clear(burst);

  struct rte_mbuf** frames = FaceRxBurst_GetScratch(burst);
  for (uint16_t i = 0; i < nFrames; ++i) {
    struct rte_mbuf* frame = frames[i];
    Face* face = Face_Get_(frame->port);
    if (unlikely(face->impl == NULL)) {
      rte_pktmbuf_free(frame);
      continue;
    }

    Packet* npkt = RxProc_Input(&face->impl->rx, rxThread, frame);
    if (npkt == NULL) {
      continue;
    }

    L3PktType l3type = Packet_GetL3PktType(npkt);
    switch (l3type) {
      case L3PktTypeInterest:
        FaceRxBurst_PutInterest(burst, npkt);
        break;
      case L3PktTypeData:
        FaceRxBurst_PutData(burst, npkt);
        break;
      case L3PktTypeNack:
        FaceRxBurst_PutNack(burst, npkt);
        break;
      default:
        assert(false);
        break;
    }
  }

  if (likely(burst->nInterests + burst->nData + burst->nNacks > 0)) {
    cb(burst, cbarg);
  }
}
