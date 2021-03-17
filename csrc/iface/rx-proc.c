#include "rx-proc.h"
#include "../core/logger.h"
#include "faceid.h"

N_LOG_INIT(RxProc);

Packet*
RxProc_Input(RxProc* rx, int thread, struct rte_mbuf* frame)
{
  FaceID faceID = frame->port;
  NDNDPDK_ASSERT(faceID != MBUF_INVALID_PORT);
  RxProcThread* rxt = &rx->threads[thread];
  rxt->nFrames[0] += frame->pkt_len;

  Packet* npkt = Packet_FromMbuf(frame);
  if (unlikely(!Packet_Parse(npkt))) {
    ++rxt->nDecodeErr;
    N_LOGD("l2-decode-error face=%" PRI_FaceID " thread=%d", faceID, thread);
    rte_pktmbuf_free(frame);
    return NULL;
  }

  PktType pktType = Packet_GetType(npkt);
  if (likely(pktType != PktFragment)) {
    ++rxt->nFrames[pktType];
    return npkt;
  }

  if (unlikely(thread != 0)) {
    // reassembler is available on thread 0 only
    N_LOGW("lp-reassembler-unavail face=%" PRI_FaceID " thread=%d", faceID, thread);
    rte_pktmbuf_free(frame);
    return NULL;
  }

  npkt = Reassembler_Accept(&rx->reass, npkt);
  frame = NULL; // disallow further usage of 'frame'
  if (npkt == NULL) {
    return NULL;
  }

  if (unlikely(!Packet_ParseL3(npkt))) {
    ++rxt->nDecodeErr;
    N_LOGD("l3-decode-error face=%" PRI_FaceID " thread=%d", faceID, thread);
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return NULL;
  }

  pktType = Packet_GetType(npkt);
  ++rxt->nFrames[pktType];
  return npkt;
}
