#include "rx-proc.h"

#include "../core/logger.h"

INIT_ZF_LOG(RxProc);

static struct rte_mbuf*
RxProc_ProcessInterest(RxProc* rx, struct rte_mbuf* pkt, TlvDecodePos* d,
                       L3PktType l3type)
{
  Packet_SetL3PktType(pkt, l3type);
  InterestPkt* interest = Packet_GetInterestHdr(pkt);
  NdnError e = DecodeInterest(d, interest);

  if (likely(e == NdnError_OK)) {
    ++rx->nFrames[l3type];
    return pkt;
  }

  ++rx->nL3DecodeErr;
  ZF_LOGD("%" PRIu16 " interest-decode-error=%d", pkt->port, e);
  rte_pktmbuf_free(pkt);
  return NULL;
}

static struct rte_mbuf*
RxProc_ProcessData(RxProc* rx, struct rte_mbuf* pkt, TlvDecodePos* d)
{
  Packet_SetL3PktType(pkt, L3PktType_Data);
  DataPkt* data = Packet_GetDataHdr(pkt);
  NdnError e = DecodeData(d, data);

  if (likely(e == NdnError_OK)) {
    ++rx->nFrames[L3PktType_Data];
    return pkt;
  }

  ++rx->nL3DecodeErr;
  ZF_LOGD("%" PRIu16 " data-decode-error=%d", pkt->port, e);
  rte_pktmbuf_free(pkt);
  return NULL;
}

// interestL3type: L3 type (Interest or Nack) if the packet is an Interest
static struct rte_mbuf*
RxProc_ProcessNetPkt(RxProc* rx, struct rte_mbuf* pkt, TlvDecodePos* d,
                     uint8_t firstOctet, L3PktType interestL3type)
{
  if (firstOctet == TT_Interest) {
    return RxProc_ProcessInterest(rx, pkt, d, interestL3type);
  }
  if (firstOctet == TT_Data) {
    return RxProc_ProcessData(rx, pkt, d);
  }

  ++rx->nL3DecodeErr;
  ZF_LOGD("%" PRIu16 " unknown-net-type=%" PRIX8, pkt->port, firstOctet);
  return NULL;
}

static struct rte_mbuf*
RxProc_ProcessLpPkt(RxProc* rx, struct rte_mbuf* pkt, TlvDecodePos* d)
{
  Packet_SetL2PktType(pkt, L2PktType_NdnlpV2);
  LpPkt* lpp = Packet_GetLpHdr(pkt);
  NdnError e = DecodeLpPkt(d, lpp);
  if (unlikely(e != NdnError_OK)) {
    ++rx->nL2DecodeErr;
    ZF_LOGD("%" PRIu16 " lp-decode-error=%d", pkt->port, e);
    rte_pktmbuf_free(pkt);
    return NULL;
  }

  if (!LpPkt_HasPayload(lpp)) {
    ZF_LOGD("%" PRIu16 " lp-no-payload", pkt->port);
    rte_pktmbuf_free(pkt);
    return NULL;
  }

  Packet_Adj(pkt, lpp->payloadOff);

  if (LpPkt_IsFragmented(lpp)) {
    pkt = InOrderReassembler_Receive(&rx->reassembler, pkt);
    if (pkt == NULL) {
      return NULL;
    }
    lpp = Packet_GetLpHdr(pkt);
  }

  TlvDecodePos d1;
  MbufLoc_Init(&d1, pkt);
  return RxProc_ProcessNetPkt(rx, pkt, &d1, MbufLoc_PeekOctet(&d1),
                              lpp->nackReason > 0 ? L3PktType_Nack
                                                  : L3PktType_Interest);
}

struct rte_mbuf*
RxProc_Input(RxProc* rx, struct rte_mbuf* frame)
{
  ++rx->nFrames[L3PktType_None];
  rx->nOctets += frame->pkt_len;

  TlvDecodePos d;
  MbufLoc_Init(&d, frame);
  uint8_t firstOctet = MbufLoc_PeekOctet(&d);

  if (firstOctet == TT_LpPacket) {
    return RxProc_ProcessLpPkt(rx, frame, &d);
  }
  return RxProc_ProcessNetPkt(rx, frame, &d, firstOctet, L3PktType_Interest);
}

void
RxProc_ReadCounters(RxProc* rx, FaceCounters* cnt)
{
  cnt->rxl2.nFrames = rx->nFrames[L3PktType_None];
  cnt->rxl2.nOctets = rx->nOctets;

  cnt->rxl2.nReassGood = rx->reassembler.nDelivered;
  cnt->rxl2.nReassBad = rx->reassembler.nIncomplete;

  cnt->rxl3.nInterests = rx->nFrames[L3PktType_Interest];
  cnt->rxl3.nData = rx->nFrames[L3PktType_Data];
  cnt->rxl3.nNacks = rx->nFrames[L3PktType_Nack];
}
