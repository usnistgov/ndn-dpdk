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
RxProc_Input(RxProc* rx, struct rte_mbuf* frame)
{
  FaceId faceId = frame->port;
  assert(faceId != MBUF_INVALID_PORT);
  ++rx->nFrames[L3PktType_None];
  rx->nOctets += frame->pkt_len;

  Packet* npkt = Packet_FromMbuf(frame);
  NdnError e = Packet_ParseL2(npkt);
  if (unlikely(e != NdnError_OK)) {
    ++rx->nL2DecodeErr;
    ZF_LOGD("%" PRI_FaceId " lp-decode-error=%d", faceId, e);
    rte_pktmbuf_free(frame);
    return NULL;
  }

  if (unlikely(frame->pkt_len == 0)) {
    ZF_LOGD("%" PRI_FaceId " lp-no-payload", faceId);
    rte_pktmbuf_free(frame);
    return NULL;
  }

  if (Packet_GetLpHdr(npkt)->l2.fragCount > 1) {
    npkt = InOrderReassembler_Receive(&rx->reassembler, npkt);
    if (npkt == NULL) {
      return NULL;
    }
    frame = NULL; // disallow further usage of 'frame'
  }

  e = Packet_ParseL3(npkt, rx->nameMp);
  if (unlikely(e != NdnError_OK)) {
    ++rx->nL3DecodeErr;
    ZF_LOGD("%" PRI_FaceId " l3-decode-error=%d", faceId, e);
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return NULL;
  }

  L3PktType l3type = Packet_GetL3PktType(npkt);
  ++rx->nFrames[l3type];
  return npkt;
}
