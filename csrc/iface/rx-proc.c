#include "rx-proc.h"
#include "../core/logger.h"
#include "faceid.h"

INIT_ZF_LOG(RxProc);

Packet*
RxProc_Input(RxProc* rx, int thread, struct rte_mbuf* frame)
{
  FaceID faceID = frame->port;
  NDNDPDK_ASSERT(faceID != MBUF_INVALID_PORT);
  RxProcThread* rxt = &rx->threads[thread];
  rxt->nOctets += frame->pkt_len;

  Packet* npkt = Packet_FromMbuf(frame);
  if (unlikely(!Packet_Parse(npkt))) {
    ++rxt->nDecodeErr;
    ZF_LOGD("%" PRI_FaceID "-%d decode-error", faceID, thread);
    rte_pktmbuf_free(frame);
    return NULL;
  }

  PktType pktType = Packet_GetType(npkt);
  if (likely(pktType != PktFragment)) {
    ++rxt->nFrames[pktType];
    return npkt;
  }

  if (unlikely(thread != 0)) {
    // currently reassembler is available on thread 0 only
    ZF_LOGW("%" PRI_FaceID "-%d lp-reassembler-unavail", faceID, thread);
    rte_pktmbuf_free(frame);
    return NULL;
  }

  npkt = InOrderReassembler_Receive(&rx->reassembler, npkt);
  frame = NULL; // disallow further usage of 'frame'
  if (npkt == NULL) {
    ++rxt->nFrames[PktFragment];
    return NULL;
  }

  if (unlikely(!Packet_ParseL3(npkt))) {
    ++rxt->nDecodeErr;
    ZF_LOGD("%" PRI_FaceID "-%d decode-error", faceID, thread);
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return NULL;
  }

  pktType = Packet_GetType(npkt);
  ++rxt->nFrames[pktType];
  return npkt;
}
