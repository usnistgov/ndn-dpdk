#include "tx-proc.h"

#include "../core/logger.h"
#include "../ndni/tlv-decoder.h"

INIT_ZF_LOG(TxProc);

__attribute__((nonnull)) static uint16_t
TxProc_OutputNoFrag(TxProc* tx, Packet* npkt, struct rte_mbuf** frames)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  NDNDPDK_ASSERT(pkt->pkt_len > 0);

  struct rte_mbuf* frame;
  if (unlikely(RTE_MBUF_CLONED(pkt) || rte_mbuf_refcnt_read(pkt) > 1 ||
               rte_pktmbuf_headroom(pkt) < RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom)) {
    frame = rte_pktmbuf_alloc(tx->headerMp);
    if (unlikely(frame == NULL)) {
      ++tx->nAllocFails;
      rte_pktmbuf_free(pkt);
      return 0;
    }
    frame->data_off = RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom;

    if (unlikely(!Mbuf_Chain(frame, frame, pkt))) {
      ++tx->nL3OverLength;
      struct rte_mbuf* frees[] = { frame, pkt };
      rte_pktmbuf_free_bulk(frees, RTE_DIM(frees));
      return 0;
    }

    Packet_SetType(Packet_FromMbuf(frame), PktType_ToSlim(Packet_GetType(npkt)));
    ZF_LOGV("pktLen=%" PRIu32 " one-fragment(alloc)", pkt->pkt_len);
  } else {
    frame = pkt;
    ZF_LOGV("pktLen=%" PRIu32 " one-fragment(prepend)", pkt->pkt_len);
  }

  LpL2 l2 = { .fragCount = 1 };
  LpHeader_Prepend(frame, Packet_GetLpL3Hdr(npkt), &l2);
  frames[0] = frame;
  return 1;
}

uint16_t
TxProc_Output(TxProc* tx, Packet* npkt, struct rte_mbuf* frames[LpMaxFragments])
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  NDNDPDK_ASSERT(pkt->pkt_len > 0);
  if (likely(pkt->pkt_len <= tx->fragmentPayloadSize)) {
    return TxProc_OutputNoFrag(tx, npkt, frames);
  }

  uint32_t fragCount = DIV_CEIL(pkt->pkt_len, tx->fragmentPayloadSize);
  ZF_LOGV("pktLen=%" PRIu32 " fragCount=%" PRIu32 " seq=%" PRIu64, pkt->pkt_len, fragCount,
          tx->nextSeqNum);
  if (unlikely(fragCount > LpMaxFragments)) {
    ++tx->nL3OverLength;
    return 0;
  }

  int res = rte_pktmbuf_alloc_bulk(tx->headerMp, frames, fragCount);
  if (unlikely(res != 0)) {
    ++tx->nAllocFails;
    rte_pktmbuf_free(pkt);
    return 0;
  }

  TlvDecoder d;
  TlvDecoder_Init(&d, pkt);
  LpL2 l2 = { .seqNumBase = tx->nextSeqNum, .fragCount = fragCount };
  LpL3* l3 = Packet_GetLpL3Hdr(npkt);

  PktType framePktType = PktType_ToSlim(Packet_GetType(npkt));

  for (l2.fragIndex = 0; l2.fragIndex < fragCount; ++l2.fragIndex) {
    uint32_t fragSize = RTE_MIN(tx->fragmentPayloadSize, d.length);
    struct rte_mbuf* payload = TlvDecoder_Clone(&d, fragSize, tx->indirectMp, NULL);
    if (unlikely(payload == NULL)) {
      NDNDPDK_ASSERT(rte_errno == ENOENT);
      ++tx->nAllocFails;
      rte_pktmbuf_free_bulk(frames, fragCount);
      rte_pktmbuf_free(pkt);
      return 0;
    }

    struct rte_mbuf* frame = frames[l2.fragIndex];
    frame->data_off = RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom;

    if (unlikely(!Mbuf_Chain(frame, frame, payload))) {
      ++tx->nL3OverLength;
      rte_pktmbuf_free_bulk(frames, fragCount);
      struct rte_mbuf* frees[] = { payload, pkt };
      rte_pktmbuf_free_bulk(frees, RTE_DIM(frees));
      return 0;
    }
    LpHeader_Prepend(frame, l3, &l2);

    // Set real L3 type on first L2 frame and None on other L2 frames,
    // to match counting logic in TxProc_CountSent
    Packet_SetType(Packet_FromMbuf(frame), framePktType);
    framePktType = PktFragment;
    frame->timestamp = pkt->timestamp;
  }
  rte_pktmbuf_free(pkt);

  ++tx->nL3Fragmented;
  tx->nextSeqNum += fragCount;
  return fragCount;
}

void
TxProc_Init(TxProc* tx, uint16_t mtu, struct rte_mempool* indirectMp, struct rte_mempool* headerMp)
{
  NDNDPDK_ASSERT(mtu >= MinMTU && mtu <= MaxMTU);
  static_assert((int)MinMTU > (int)LpHeaderHeadroom, "");
  NDNDPDK_ASSERT(rte_pktmbuf_data_room_size(headerMp) >= RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom);
  tx->indirectMp = indirectMp;
  tx->headerMp = headerMp;

  tx->fragmentPayloadSize = mtu - LpHeaderHeadroom;
}
