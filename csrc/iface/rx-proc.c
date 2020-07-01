#include "rx-proc.h"
#include "../core/logger.h"
#include "faceid.h"

INIT_ZF_LOG(RxProc);

int
RxProc_Init(RxProc* rx, struct rte_mempool* nameMp)
{
  rx->nameMp = nameMp;
  return 0;
}

Packet*
RxProc_Input(RxProc* rx, int thread, struct rte_mbuf* frame)
{
  FaceID faceID = frame->port;
  assert(faceID != MBUF_INVALID_PORT);
  RxProcThread* rxt = &rx->threads[thread];
  ++rxt->nFrames[L3PktTypeNone];
  rxt->nOctets += frame->pkt_len;
  frame->packet_type = 0;

  Packet* npkt = Packet_FromMbuf(frame);
  NdnError e = Packet_ParseL2(npkt);
  if (unlikely(e != NdnErrOK)) {
    ++rxt->nL2DecodeErr;
    ZF_LOGD("%" PRI_FaceID "-%d lp-decode-error=%d", faceID, thread, e);
    rte_pktmbuf_free(frame);
    return NULL;
  }

  if (unlikely(frame->pkt_len == 0)) {
    ZF_LOGD("%" PRI_FaceID "-%d lp-no-payload", faceID, thread);
    rte_pktmbuf_free(frame);
    return NULL;
  }

  if (Packet_GetLpHdr(npkt)->l2.fragCount > 1) {
    if (unlikely(thread != 0)) {
      // currently reassembler is available on thread 0 only
      ZF_LOGW("%" PRI_FaceID "-%d lp-reassembler-unavail", faceID, thread);
      rte_pktmbuf_free(frame);
      return NULL;
    }
    npkt = InOrderReassembler_Receive(&rx->reassembler, npkt);
    if (npkt == NULL) {
      return NULL;
    }
    frame = NULL; // disallow further usage of 'frame'
  }

  e = Packet_ParseL3(npkt, rx->nameMp);
  if (unlikely(e != NdnErrOK)) {
    ++rxt->nL3DecodeErr;
    ZF_LOGD("%" PRI_FaceID "-%d l3-decode-error=%d", faceID, thread, e);
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return NULL;
  }

  L3PktType l3type = Packet_GetL3PktType(npkt);
  ++rxt->nFrames[l3type];
  return npkt;
}
