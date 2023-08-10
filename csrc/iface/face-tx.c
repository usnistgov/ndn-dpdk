#include "face-impl.h"

#include "../core/logger.h"
#include "../ndni/tlv-decoder.h"

static_assert((int)MinMTU > (int)LpHeaderHeadroom, "");

N_LOG_INIT(FaceTx);

__attribute__((nonnull)) static __rte_always_inline uint16_t
FaceTx_One(const char* logVerb, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments]) {
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  N_LOGV("%s pktLen=%" PRIu32, logVerb, pkt->pkt_len);

  LpL2 l2 = {.fragCount = 1};
  LpHeader_Prepend(pkt, Packet_GetLpL3Hdr(npkt), &l2);
  frames[0] = pkt;
  return 1;
}

__attribute__((nonnull)) static uint16_t
FaceTx_LinearOne(Face* face, int txThread, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments]) {
  return FaceTx_One("linear-one", npkt, frames);
}

__attribute__((nonnull)) static uint16_t
FaceTx_ChainedOne(Face* face, int txThread, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments]) {
  return FaceTx_One("chained-one", npkt, frames);
}

__attribute__((nonnull)) static uint16_t
FaceTx_LinearFrag(Face* face, int txThread, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments]) {
  FaceTxThread* txt = &face->impl->tx[txThread];
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  NDNDPDK_ASSERT(pkt->nb_segs > 1); // single-fragment packet should invoke FaceTx_LinearOne
  LpL2 l2 = {.seqNumBase = txt->nextSeqNum, .fragCount = pkt->nb_segs};
  N_LOGV("linear-frag pktLen=%" PRIu32 " seq=%016" PRIx64 " fragCount=%" PRIu8, pkt->pkt_len,
         l2.seqNumBase, l2.fragCount);
  LpL3* l3 = Packet_GetLpL3Hdr(npkt);
  TscTime timestamp = Mbuf_GetTimestamp(pkt);
  PktType framePktType = PktType_ToSlim(Packet_GetType(npkt));

  for (l2.fragIndex = 0; l2.fragIndex < l2.fragCount; ++l2.fragIndex) {
    NDNDPDK_ASSERT(pkt != NULL);

    struct rte_mbuf* next = pkt->next;
    pkt->next = NULL;
    pkt->nb_segs = 1;
    pkt->pkt_len = pkt->data_len;

    FaceTx_CheckDirectFragmentMbuf_(pkt);
    NDNDPDK_ASSERT(pkt->pkt_len <= face->txAlign.fragmentPayloadSize);

    LpHeader_Prepend(pkt, l3, &l2);
    Mbuf_SetTimestamp(pkt, timestamp);
    Packet_SetType(Packet_FromMbuf(pkt), framePktType);
    framePktType = PktFragment;

    frames[l2.fragIndex] = pkt;
    pkt = next;
  }

  ++txt->nL3Fragmented;
  txt->nextSeqNum += l2.fragCount;
  return l2.fragCount;
}

__attribute__((nonnull)) static uint16_t
FaceTx_ChainedFrag(Face* face, int txThread, Packet* npkt,
                   struct rte_mbuf* frames[LpMaxFragments]) {
  FaceTxThread* txt = &face->impl->tx[txThread];
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  LpL2 l2 = {.seqNumBase = txt->nextSeqNum};
  l2.fragCount = SPDK_CEIL_DIV(pkt->pkt_len, face->txAlign.fragmentPayloadSize);
  N_LOGV("chained-frag pktLen=%" PRIu32 " seq=%016" PRIx64 " fragCount=%" PRIu8, pkt->pkt_len,
         l2.seqNumBase, l2.fragCount);
  if (unlikely(l2.fragCount > LpMaxFragments)) {
    ++txt->nL3OverLength;
    return 0;
  }
  if (unlikely(rte_pktmbuf_alloc_bulk(face->impl->txMempools.header, frames, l2.fragCount) != 0)) {
    ++txt->nAllocFails;
    rte_pktmbuf_free(pkt);
    return 0;
  }

  LpL3* l3 = Packet_GetLpL3Hdr(npkt);
  TscTime timestamp = Mbuf_GetTimestamp(pkt);
  PktType framePktType = PktType_ToSlim(Packet_GetType(npkt));
  TlvDecoder d = TlvDecoder_Init(pkt);

  for (l2.fragIndex = 0; l2.fragIndex < l2.fragCount; ++l2.fragIndex) {
    struct rte_mbuf* frame = frames[l2.fragIndex];
    frame->data_off = RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom;

    uint32_t fragSize = RTE_MIN(face->txAlign.fragmentPayloadSize, d.length);
    struct rte_mbuf* payload = TlvDecoder_Clone(&d, fragSize, face->impl->txMempools.indirect);
    if (unlikely(payload == NULL)) {
      ++txt->nAllocFails;
      rte_pktmbuf_free_bulk(frames, l2.fragCount);
      rte_pktmbuf_free(pkt);
      return 0;
    }

    if (unlikely(!Mbuf_Chain(frame, frame, payload))) {
      ++txt->nL3OverLength;
      frames[l2.fragIndex] = NULL; // frame is freed by Mbuf_Chain
      rte_pktmbuf_free_bulk(frames, l2.fragCount);
      rte_pktmbuf_free(pkt);
      return 0;
    }

    LpHeader_Prepend(frame, l3, &l2);
    Mbuf_SetTimestamp(frame, timestamp);
    Packet_SetType(Packet_FromMbuf(frame), framePktType);
    framePktType = PktFragment;
  }

  rte_pktmbuf_free(pkt);
  ++txt->nL3Fragmented;
  txt->nextSeqNum += l2.fragCount;
  return l2.fragCount;
}

FaceTx_OutputFunc FaceTx_OutputJmp[] = {
  [FaceTx_OutputFuncIndex(true, true)] = FaceTx_LinearOne,
  [FaceTx_OutputFuncIndex(true, false)] = FaceTx_LinearFrag,
  [FaceTx_OutputFuncIndex(false, true)] = FaceTx_ChainedOne,
  [FaceTx_OutputFuncIndex(false, false)] = FaceTx_ChainedFrag,
};
