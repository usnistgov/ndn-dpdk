#include "tx-proc.h"

#include "../core/logger.h"

// minimum payload size per fragment
static const int MIN_PAYLOAD_SIZE_PER_FRAGMENT = 512;

uint16_t TxProc_OutputFrag(TxProc* tx, struct rte_mbuf* pkt,
                           struct rte_mbuf** frames, uint16_t maxFrames);
uint16_t TxProc_OutputNoFrag(TxProc* tx, struct rte_mbuf* pkt,
                             struct rte_mbuf** frames, uint16_t maxFrames);

int
TxProc_Init(TxProc* tx, uint16_t mtu, uint16_t headroom,
            struct rte_mempool* indirectMp, struct rte_mempool* headerMp)
{
  assert(indirectMp != NULL);
  assert(headerMp != NULL);
  assert(rte_pktmbuf_data_room_size(headerMp) >=
         EncodeLpHeaders_GetHeadroom() + EncodeLpHeaders_GetTailroom());
  tx->indirectMp = indirectMp;
  tx->headerMp = headerMp;

  if (mtu == 0) {
    tx->outputFunc = TxProc_OutputNoFrag;
  } else {
    int fragmentPayloadSize =
      (int)mtu - EncodeLpHeaders_GetHeadroom() - EncodeLpHeaders_GetTailroom();
    if (fragmentPayloadSize < MIN_PAYLOAD_SIZE_PER_FRAGMENT) {
      return ENOSPC;
    }
    tx->fragmentPayloadSize = (uint16_t)fragmentPayloadSize;
    tx->outputFunc = TxProc_OutputFrag;
  }

  tx->headerHeadroom = headroom + EncodeLpHeaders_GetHeadroom();
  if (rte_pktmbuf_data_room_size(headerMp) <
      tx->headerHeadroom + EncodeLpHeaders_GetTailroom()) {
    return ERANGE;
  }

  return 0;
}

static inline void
TxProc_PrepareLpPkt(TxProc* tx, struct rte_mbuf* pkt, LpPkt* lpp)
{
  if (Packet_GetL2PktType(pkt) == L2PktType_NdnlpV2) {
    rte_memcpy(lpp, Packet_GetLpHdr(pkt), sizeof(*lpp));
  } else {
    memset(lpp, 0, sizeof(*lpp));
  }
}

uint16_t
TxProc_OutputFrag(TxProc* tx, struct rte_mbuf* pkt, struct rte_mbuf** frames,
                  uint16_t maxFrames)
{
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

  LpPkt lpp;
  TxProc_PrepareLpPkt(tx, pkt, &lpp);
  MbufLoc pos;
  MbufLoc_Init(&pos, pkt);
  NdnPktType l3type = Packet_GetNdnPktType(pkt);

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
    MbufLoc_Init(&lpp.payload, payload);

    lpp.seqNo = ++tx->lastSeqNo;
    lpp.fragIndex = (uint16_t)i;
    lpp.fragCount = (uint16_t)nFragments;

    struct rte_mbuf* frame = frames[i];
    frame->data_off = tx->headerHeadroom;
    EncodeLpHeaders(frame, &lpp);
    res = rte_pktmbuf_chain(frame, payload);
    if (unlikely(res != 0)) {
      ++tx->nL3OverLength;
      FreeMbufs(frames, nFragments);
      rte_pktmbuf_free(payload);
      return 0;
    }

    // Set real L3 type on first segment and None on other segments,
    // to match counting logic in TxProc_Sent
    Packet_SetNdnPktType(frame, l3type);
    l3type = NdnPktType_None;
  }

  ++tx->nL3Pkts[(int)(nFragments > 1)];
  return nFragments;
}

uint16_t
TxProc_OutputNoFrag(TxProc* tx, struct rte_mbuf* pkt, struct rte_mbuf** frames,
                    uint16_t maxFrames)
{
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

  LpPkt lpp;
  TxProc_PrepareLpPkt(tx, pkt, &lpp);
  lpp.fragIndex = 0;
  lpp.fragCount = 1;
  MbufLoc_Init(&lpp.payload, payload);

  frame->data_off = tx->headerHeadroom;
  EncodeLpHeaders(frame, &lpp);
  int res = rte_pktmbuf_chain(frame, payload);
  if (unlikely(res != 0)) {
    ++tx->nL3OverLength;
    rte_pktmbuf_free(frame);
    rte_pktmbuf_free(payload);
    return 0;
  }

  ++tx->nL3Pkts[0];
  return 1;
}

void
TxProc_ReadCounters(TxProc* tx, FaceCounters* cnt)
{
  cnt->txl2.nOctets = tx->nOctets;
  cnt->txl2.nFragGood = tx->nL3Pkts[1];
  cnt->txl2.nFragBad = tx->nL3OverLength + tx->nAllocFails;

  cnt->txl3.nInterests = tx->nFrames[NdnPktType_Interest];
  cnt->txl3.nData = tx->nFrames[NdnPktType_Data];
  cnt->txl3.nNacks = tx->nFrames[NdnPktType_Nack];

  cnt->txl2.nFrames = tx->nFrames[NdnPktType_None] + cnt->txl3.nInterests +
                      cnt->txl3.nData + cnt->txl3.nNacks;
}
