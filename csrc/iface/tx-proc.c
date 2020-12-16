#include "tx-proc.h"

#include "../core/logger.h"
#include "../ndni/tlv-decoder.h"

INIT_ZF_LOG(TxProc);

static_assert((int)MinMTU > (int)LpHeaderHeadroom, "");

__attribute__((nonnull)) static __rte_always_inline uint16_t
TxProc_One(const char* logVerb, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments])
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  ZF_LOGV("%s pktLen=%" PRIu32, logVerb, pkt->pkt_len);

  LpL2 l2 = { .fragCount = 1 };
  LpHeader_Prepend(pkt, Packet_GetLpL3Hdr(npkt), &l2);
  frames[0] = pkt;
  return 1;
}

__attribute__((nonnull)) static uint16_t
TxProc_LinearOne(TxProc* tx, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments],
                 PacketTxAlign align)
{
  return TxProc_One("linear-one", npkt, frames);
}

__attribute__((nonnull)) static uint16_t
TxProc_ChainedOne(TxProc* tx, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments],
                  PacketTxAlign align)
{
  return TxProc_One("chained-one", npkt, frames);
}

__attribute__((nonnull)) static uint16_t
TxProc_LinearFrag(TxProc* tx, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments],
                  PacketTxAlign align)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  LpL2 l2 = { .seqNumBase = tx->nextSeqNum, .fragCount = pkt->nb_segs };
  ZF_LOGV("linear-frag pktLen=%" PRIu32 " seq=%016" PRIx64 " fragCount=%" PRIu8, pkt->pkt_len,
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

    TxProc_CheckDirectFragmentMbuf_(pkt);
    NDNDPDK_ASSERT(pkt->pkt_len <= align.fragmentPayloadSize);

    LpHeader_Prepend(pkt, l3, &l2);
    Mbuf_SetTimestamp(pkt, timestamp);
    Packet_SetType(Packet_FromMbuf(pkt), framePktType);
    framePktType = PktFragment;

    frames[l2.fragIndex] = pkt;
    pkt = next;
  }

  ++tx->nL3Fragmented;
  tx->nextSeqNum += l2.fragCount;
  return l2.fragCount;
}

__attribute__((nonnull)) static uint16_t
TxProc_ChainedFrag(TxProc* tx, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments],
                   PacketTxAlign align)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  LpL2 l2 = { .seqNumBase = tx->nextSeqNum };
  l2.fragCount = DIV_CEIL(pkt->pkt_len, align.fragmentPayloadSize);
  ZF_LOGV("chained-frag pktLen=%" PRIu32 " seq=%016" PRIx64 " fragCount=%" PRIu8, pkt->pkt_len,
          l2.seqNumBase, l2.fragCount);
  if (unlikely(l2.fragCount > LpMaxFragments)) {
    ++tx->nL3OverLength;
    return 0;
  }
  if (unlikely(rte_pktmbuf_alloc_bulk(tx->mp.header, frames, l2.fragCount) != 0)) {
    ++tx->nAllocFails;
    rte_pktmbuf_free(pkt);
    return 0;
  }

  LpL3* l3 = Packet_GetLpL3Hdr(npkt);
  TscTime timestamp = Mbuf_GetTimestamp(pkt);
  PktType framePktType = PktType_ToSlim(Packet_GetType(npkt));
  TlvDecoder d;
  TlvDecoder_Init(&d, pkt);

  for (l2.fragIndex = 0; l2.fragIndex < l2.fragCount; ++l2.fragIndex) {
    struct rte_mbuf* frame = frames[l2.fragIndex];
    frame->data_off = RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom;

    uint32_t fragSize = RTE_MIN(align.fragmentPayloadSize, d.length);
    struct rte_mbuf* payload = TlvDecoder_Clone(&d, fragSize, tx->mp.indirect, NULL);
    if (unlikely(payload == NULL)) {
      ++tx->nAllocFails;
      rte_pktmbuf_free_bulk(frames, l2.fragCount);
      rte_pktmbuf_free(pkt);
      return 0;
    }

    if (unlikely(!Mbuf_Chain(frame, frame, payload))) {
      ++tx->nL3OverLength;
      rte_pktmbuf_free_bulk(frames, l2.fragCount);
      rte_pktmbuf_free(payload);
      rte_pktmbuf_free(pkt);
      return 0;
    }

    LpHeader_Prepend(frame, l3, &l2);
    Mbuf_SetTimestamp(frame, timestamp);
    Packet_SetType(Packet_FromMbuf(frame), framePktType);
    framePktType = PktFragment;
  }

  rte_pktmbuf_free(pkt);
  ++tx->nL3Fragmented;
  tx->nextSeqNum += l2.fragCount;
  return l2.fragCount;
}

__attribute__((nonnull)) void
TxProc_Init(TxProc* tx, PacketTxAlign align)
{
  if (align.linearize) {
    tx->outputFunc[0] = TxProc_LinearFrag;
    tx->outputFunc[1] = TxProc_LinearOne;
  } else {
    tx->outputFunc[0] = TxProc_ChainedFrag;
    tx->outputFunc[1] = TxProc_ChainedOne;
  }
}
