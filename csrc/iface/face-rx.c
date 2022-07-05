#include "face-impl.h"

#include "../core/logger.h"

N_LOG_INIT(FaceRx);

Packet*
FaceRx_Input(Face* face, int rxThread, struct rte_mbuf* pkt)
{
  NDNDPDK_ASSERT(pkt->port == face->id);
  FaceRxThread* rxt = &face->impl->rx[rxThread];
  rxt->nFrames[0] += pkt->pkt_len; // nOctets counter

  Packet* npkt = Packet_FromMbuf(pkt);
  if (unlikely(!Packet_Parse(npkt, face->impl->rxParseFor))) {
    ++rxt->nDecodeErr;
    N_LOGD("l2-decode-error face=%" PRI_FaceID " thread=%d", face->id, rxThread);
    rte_pktmbuf_free(pkt);
    return NULL;
  }

  PktType pktType = Packet_GetType(npkt);
  if (likely(pktType != PktFragment)) {
    ++rxt->nFrames[pktType];
    return npkt;
  }

  NULLize(pkt); // pkt aliases npkt, but npkt will be owned by reassembler
  npkt = Reassembler_Accept(&rxt->reass, npkt);
  if (npkt == NULL) {
    return NULL;
  }

  if (unlikely(!Packet_ParseL3(npkt, face->impl->rxParseFor))) {
    ++rxt->nDecodeErr;
    N_LOGD("l3-decode-error face=%" PRI_FaceID " thread=%d", face->id, rxThread);
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return NULL;
  }

  pktType = Packet_GetType(npkt);
  ++rxt->nFrames[pktType];
  return npkt;
}
