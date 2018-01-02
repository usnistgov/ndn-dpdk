#include "rx-proc.h"

#include "../core/logger.h"

static inline struct rte_mbuf*
RxProc_ProcessInterest(RxProc* rx, struct rte_mbuf* pkt, TlvDecoder* d)
{
  // TODO distinguish Nack from Interest
  Packet_SetNdnPktType(pkt, NdnPktType_Interest);
  InterestPkt* interest = Packet_GetInterestHdr(pkt);
  NdnError e = DecodeInterest(d, interest);

  if (likely(e == NdnError_OK)) {
    ++rx->nInterests;
    return pkt;
  }

  ++rx->nL3DecodeErr;
  ZF_LOGD("%" PRIu16 " interest-decode-error=%d", pkt->port, e);
  rte_pktmbuf_free(pkt);
  return NULL;
}

static inline struct rte_mbuf*
RxProc_ProcessData(RxProc* rx, struct rte_mbuf* pkt, TlvDecoder* d)
{
  Packet_SetNdnPktType(pkt, NdnPktType_Data);
  DataPkt* data = Packet_GetDataHdr(pkt);
  NdnError e = DecodeData(d, data);

  if (likely(e == NdnError_OK)) {
    ++rx->nData;
    return pkt;
  }

  ++rx->nL3DecodeErr;
  ZF_LOGD("%" PRIu16 " data-decode-error=%d", pkt->port, e);
  rte_pktmbuf_free(pkt);
  return NULL;
}

static inline struct rte_mbuf*
RxProc_ProcessNetPkt(RxProc* rx, struct rte_mbuf* pkt, TlvDecoder* d,
                     uint8_t firstOctet)
{
  if (firstOctet == TT_Interest) {
    return RxProc_ProcessInterest(rx, pkt, d);
  }
  if (firstOctet == TT_Data) {
    return RxProc_ProcessData(rx, pkt, d);
  }

  ++rx->nL3DecodeErr;
  ZF_LOGD("%" PRIu16 " unknown-net-type=%" PRIX8, pkt->port, firstOctet);
  return NULL;
}

static inline struct rte_mbuf*
RxProc_ProcessLpPkt(RxProc* rx, struct rte_mbuf* pkt, TlvDecoder* d)
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
    lpp = NULL; // received lpp does not apply to reassembled packet
    pkt = InOrderReassembler_Receive(&rx->reassembler, pkt);
    if (pkt == NULL) {
      return NULL;
    }
  }

  TlvDecoder d1;
  MbufLoc_Init(&d1, pkt);
  return RxProc_ProcessNetPkt(rx, pkt, &d1, MbufLoc_PeekOctet(&d1));
}

struct rte_mbuf*
RxProc_Input(RxProc* rx, struct rte_mbuf* frame)
{
  ++rx->nFrames;
  rx->nOctets += frame->pkt_len;

  TlvDecoder d;
  MbufLoc_Init(&d, frame);
  uint8_t firstOctet = MbufLoc_PeekOctet(&d);

  if (firstOctet == TT_LpPacket) {
    return RxProc_ProcessLpPkt(rx, frame, &d);
  }
  return RxProc_ProcessNetPkt(rx, frame, &d, firstOctet);
}

void
RxProc_ReadCounters(RxProc* rx, FaceCounters* cnt)
{
  cnt->rxl2.nFrames = rx->nFrames;
  cnt->rxl2.nOctets = rx->nOctets;

  cnt->rxl2.nReassGood = rx->reassembler.nDelivered;
  cnt->rxl2.nReassBad = rx->reassembler.nIncomplete;

  cnt->rxl3.nInterests = rx->nInterests;
  cnt->rxl3.nData = rx->nData;
  cnt->rxl3.nNacks = rx->nNacks;
}
