#include "face-impl.h"

#include "../core/logger.h"

N_LOG_INIT(FaceRx);

void
FaceRx_Input(Face* face, int rxThread, FaceRxInputCtx* ctx) {
  FaceRxThread* rxt = &face->impl->rx[rxThread];

  for (uint16_t i = 0; i < ctx->count; ++i) {
    struct rte_mbuf* pkt = ctx->pkts[i];
    rxt->nFrames[FaceRxThread_cntNOctets] += pkt->pkt_len;

    Packet* npkt = Packet_FromMbuf(pkt);
    if (unlikely(!Packet_Parse(npkt, face->impl->rxParseFor))) {
      ++rxt->nDecodeErr;
      N_LOGD("l2-decode-error face=%" PRI_FaceID " thread=%d", face->id, rxThread);
      ctx->frees[ctx->nFree++] = pkt;
      continue;
    }
    NULLize(pkt); // pkt aliases npkt, but npkt will be owned by reassembler

    PktType pktType = Packet_GetType(npkt);
    if (unlikely(pktType == PktFragment)) {
      npkt = Reassembler_Accept(&rxt->reass, npkt);
      if (npkt == NULL) {
        continue;
      }

      if (unlikely(!Packet_ParseL3(npkt, face->impl->rxParseFor))) {
        ++rxt->nDecodeErr;
        N_LOGD("l3-decode-error face=%" PRI_FaceID " thread=%d", face->id, rxThread);
        ctx->frees[ctx->nFree++] = Packet_ToMbuf(npkt);
        continue;
      }

      pktType = Packet_GetType(npkt);
    }

    ++rxt->nFrames[pktType];
    ctx->npkts[ctx->nL3++] = npkt;
  }
}

STATIC_ASSERT_FUNC_TYPE(Face_RxInputFunc, FaceRx_Input);
