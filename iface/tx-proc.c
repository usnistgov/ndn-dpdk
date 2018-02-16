#include "tx-proc.h"

#include "../core/logger.h"

// minimum payload size per fragment
static const int MIN_PAYLOAD_SIZE_PER_FRAGMENT = 512;

uint16_t TxProc_OutputFrag(TxProc* tx, Packet* npkt, struct rte_mbuf** frames,
                           uint16_t maxFrames);
uint16_t TxProc_OutputNoFrag(TxProc* tx, Packet* npkt, struct rte_mbuf** frames,
                             uint16_t maxFrames);

int
TxProc_Init(TxProc* tx, uint16_t mtu, uint16_t headroom,
            struct rte_mempool* indirectMp, struct rte_mempool* headerMp)
{
  assert(indirectMp != NULL);
  assert(headerMp != NULL);
  assert(rte_pktmbuf_data_room_size(headerMp) >=
         EncodeLpHeader_GetHeadroom() + EncodeLpHeader_GetTailroom());
  tx->indirectMp = indirectMp;
  tx->headerMp = headerMp;

  if (mtu == 0) {
    tx->outputFunc = TxProc_OutputNoFrag;
  } else {
    int fragmentPayloadSize =
      (int)mtu - EncodeLpHeader_GetHeadroom() - EncodeLpHeader_GetTailroom();
    if (fragmentPayloadSize < MIN_PAYLOAD_SIZE_PER_FRAGMENT) {
      return ENOSPC;
    }
    tx->fragmentPayloadSize = (uint16_t)fragmentPayloadSize;
    tx->outputFunc = TxProc_OutputFrag;
  }

  tx->headerHeadroom = headroom + EncodeLpHeader_GetHeadroom();
  if (rte_pktmbuf_data_room_size(headerMp) <
      tx->headerHeadroom + EncodeLpHeader_GetTailroom()) {
    return ERANGE;
  }

  return 0;
}

uint16_t
TxProc_OutputFrag(TxProc* tx, Packet* npkt, struct rte_mbuf** frames,
                  uint16_t maxFrames)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  assert(pkt->pkt_len > 0);
  uint16_t nFragments = pkt->pkt_len / tx->fragmentPayloadSize +
                        (uint16_t)(pkt->pkt_len % tx->fragmentPayloadSize > 0);
  if (unlikely(nFragments > maxFrames)) {
    ++tx->nL3OverLength;
    return 0;
  }

  int res = rte_pktmbuf_alloc_bulk(tx->headerMp, frames, nFragments);
  if (unlikely(res != 0)) {
    ++tx->nAllocFails;
    return 0;
  }

  MbufLoc pos;
  MbufLoc_Init(&pos, pkt);
  LpHeader lph = { 0 };
  rte_memcpy(&lph.l3, Packet_InitLpL3Hdr(npkt), sizeof(lph.l3));
  lph.l2.fragCount = (uint16_t)nFragments;
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
      return 0;
    }

    lph.l2.seqNo = ++tx->lastSeqNo;
    lph.l2.fragIndex = (uint16_t)i;

    struct rte_mbuf* frame = frames[i];
    frame->data_off = tx->headerHeadroom;
    EncodeLpHeader(frame, &lph, payload->pkt_len);
    res = rte_pktmbuf_chain(frame, payload);
    if (unlikely(res != 0)) {
      ++tx->nL3OverLength;
      FreeMbufs(frames, nFragments);
      rte_pktmbuf_free(payload);
      return 0;
    }

    // Set real L3 type on first segment and None on other segments,
    // to match counting logic in TxProc_CountSent
    frame->inner_l3_type = l3type;
    l3type = L3PktType_None;
  }

  ++tx->nL3Pkts[(int)(nFragments > 1)];
  return nFragments;
}

uint16_t
TxProc_OutputNoFrag(TxProc* tx, Packet* npkt, struct rte_mbuf** frames,
                    uint16_t maxFrames)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  assert(pkt->pkt_len > 0);
  assert(maxFrames >= 1);

  struct rte_mbuf* frame = frames[0] = rte_pktmbuf_alloc(tx->headerMp);
  if (unlikely(frame == NULL)) {
    ++tx->nAllocFails;
    return 0;
  }

  struct rte_mbuf* payload = rte_pktmbuf_clone(pkt, tx->indirectMp);
  if (unlikely(payload == NULL)) {
    assert(rte_errno == ENOENT);
    ++tx->nAllocFails;
    rte_pktmbuf_free(frame);
    return 0;
  }

  LpHeader lph = { 0 };
  rte_memcpy(&lph.l3, Packet_InitLpL3Hdr(npkt), sizeof(lph.l3));
  lph.l2.fragIndex = 0;
  lph.l2.fragCount = 1;

  frame->data_off = tx->headerHeadroom;
  EncodeLpHeader(frame, &lph, payload->pkt_len);
  int res = rte_pktmbuf_chain(frame, payload);
  if (unlikely(res != 0)) {
    ++tx->nL3OverLength;
    rte_pktmbuf_free(frame);
    rte_pktmbuf_free(payload);
    return 0;
  }

  frame->inner_l3_type = Packet_GetL3PktType(npkt);
  ++tx->nL3Pkts[0];
  return 1;
}

void
TxProc_ReadCounters(TxProc* tx, FaceCounters* cnt)
{
  cnt->txl2.nOctets = tx->nOctets;
  cnt->txl2.nFragGood = tx->nL3Pkts[1];
  cnt->txl2.nFragBad = tx->nL3OverLength + tx->nAllocFails;

  cnt->txl3.nInterests = tx->nFrames[L3PktType_Interest];
  cnt->txl3.nData = tx->nFrames[L3PktType_Data];
  cnt->txl3.nNacks = tx->nFrames[L3PktType_Nack];

  cnt->txl2.nFrames = tx->nFrames[L3PktType_None] + cnt->txl3.nInterests +
                      cnt->txl3.nData + cnt->txl3.nNacks;
}
