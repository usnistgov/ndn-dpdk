#include "tx-proc.h"

#include "../core/logger.h"

INIT_ZF_LOG(TxProc);

// minimum payload size per fragment
static const int MIN_PAYLOAD_SIZE_PER_FRAGMENT = 512;

uint16_t
TxProc_OutputFrag(TxProc* tx,
                  Packet* npkt,
                  struct rte_mbuf** frames,
                  uint16_t maxFrames);
uint16_t
TxProc_OutputNoFrag(TxProc* tx,
                    Packet* npkt,
                    struct rte_mbuf** frames,
                    uint16_t maxFrames);

int
TxProc_Init(TxProc* tx,
            uint16_t mtu,
            uint16_t headroom,
            struct rte_mempool* indirectMp,
            struct rte_mempool* headerMp)
{
  assert(indirectMp != NULL);
  assert(headerMp != NULL);
  assert(rte_pktmbuf_data_room_size(headerMp) >=
         headroom + LpHeaderEstimatedHeadroom);
  tx->indirectMp = indirectMp;
  tx->headerMp = headerMp;

  if (mtu == 0) {
    tx->outputFunc = TxProc_OutputNoFrag;
  } else {
    int fragmentPayloadSize = (int)mtu - LpHeaderEstimatedHeadroom;
    if (fragmentPayloadSize < MIN_PAYLOAD_SIZE_PER_FRAGMENT) {
      return ENOSPC;
    }
    tx->fragmentPayloadSize = (uint16_t)fragmentPayloadSize;
    tx->outputFunc = TxProc_OutputFrag;
  }

  tx->headerHeadroom = headroom + LpHeaderEstimatedHeadroom;
  return 0;
}

uint16_t
TxProc_OutputFrag(TxProc* tx,
                  Packet* npkt,
                  struct rte_mbuf** frames,
                  uint16_t maxFrames)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  assert(pkt->pkt_len > 0);
  uint16_t nFragments = pkt->pkt_len / tx->fragmentPayloadSize +
                        (uint16_t)(pkt->pkt_len % tx->fragmentPayloadSize > 0);
  if (nFragments == 1) {
    return TxProc_OutputNoFrag(tx, npkt, frames, maxFrames);
  }
  ZF_LOGV("pktLen=%" PRIu32 " nFragments=%" PRIu16 " seq=%" PRIu64,
          pkt->pkt_len,
          nFragments,
          tx->lastSeqNum + 1);
  if (unlikely(nFragments > maxFrames)) {
    ++tx->nL3OverLength;
    return 0;
  }

  int res = rte_pktmbuf_alloc_bulk(tx->headerMp, frames, nFragments);
  if (unlikely(res != 0)) {
    ++tx->nAllocFails;
    rte_pktmbuf_free(pkt);
    return 0;
  }

  MbufLoc pos;
  MbufLoc_Init(&pos, pkt);
  LpHeader lph = { .l2 = { .fragCount = (uint16_t)nFragments } };
  rte_memcpy(&lph.l3, Packet_InitLpL3Hdr(npkt), sizeof(lph.l3));
  L3PktType l3type = Packet_GetL3PktType(npkt);

  for (int i = 0; i < nFragments; ++i) {
    uint32_t fragSize = tx->fragmentPayloadSize;
    if (fragSize > pos.rem) {
      fragSize = pos.rem;
    }
    struct rte_mbuf* payload =
      MbufLoc_MakeIndirect(&pos, fragSize, tx->indirectMp);
    if (unlikely(payload == NULL)) {
      assert(rte_errno == ENOENT);
      ++tx->nAllocFails;
      FreeMbufs(frames, nFragments);
      rte_pktmbuf_free(pkt);
      return 0;
    }

    lph.l2.seqNum = ++tx->lastSeqNum;
    lph.l2.fragIndex = (uint16_t)i;

    struct rte_mbuf* frame = frames[i];
    frame->data_off = tx->headerHeadroom;
    PrependLpHeader(frame, &lph, payload->pkt_len);
    res = rte_pktmbuf_chain(frame, payload);
    if (unlikely(res != 0)) {
      ++tx->nL3OverLength;
      FreeMbufs(frames, nFragments);
      rte_pktmbuf_free(payload);
      rte_pktmbuf_free(pkt);
      return 0;
    }

    // Set real L3 type on first L2 frame and None on other L2 frames,
    // to match counting logic in TxProc_CountSent
    frame->inner_l3_type = l3type;
    l3type = L3PktTypeNone;
    frame->timestamp = pkt->timestamp;
  }
  rte_pktmbuf_free(pkt);

  ++tx->nL3Fragmented;
  return nFragments;
}

uint16_t
TxProc_OutputNoFrag(TxProc* tx,
                    Packet* npkt,
                    struct rte_mbuf** frames,
                    uint16_t maxFrames)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint32_t payloadL = pkt->pkt_len;
  assert(payloadL > 0);
  assert(maxFrames >= 1);

  struct rte_mbuf* frame;
  if (RTE_MBUF_CLONED(pkt) || pkt->refcnt > 1 ||
      rte_pktmbuf_headroom(pkt) < tx->headerHeadroom) {
    frame = rte_pktmbuf_alloc(tx->headerMp);
    if (unlikely(frame == NULL)) {
      ++tx->nAllocFails;
      rte_pktmbuf_free(pkt);
      return 0;
    }
    frame->data_off = tx->headerHeadroom;

    int res = rte_pktmbuf_chain(frame, pkt);
    if (unlikely(res != 0)) {
      ++tx->nL3OverLength;
      rte_pktmbuf_free(frame);
      rte_pktmbuf_free(pkt);
      return 0;
    }

    frame->inner_l3_type = Packet_GetL3PktType(npkt);
    ZF_LOGV("pktLen=%" PRIu32 " one-fragment(alloc)", pkt->pkt_len);
  } else {
    frame = pkt;
    ZF_LOGV("pktLen=%" PRIu32 " one-fragment(prepend)", pkt->pkt_len);
  }

  LpHeader lph = { .l2 = { .fragCount = 1 } };
  rte_memcpy(&lph.l3, Packet_InitLpL3Hdr(npkt), sizeof(lph.l3));
  PrependLpHeader(frame, &lph, payloadL);
  frames[0] = frame;
  return 1;
}
