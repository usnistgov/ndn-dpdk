#include "rx-face.h"

#include "../ndn/protonum.h"
#include <rte_ether.h>

static inline bool
RxFace_ProcessFrame(RxFace* face, struct rte_mbuf* pkt)
{
  ++face->nFrames;

  if (unlikely(pkt->pkt_len < sizeof(struct ether_hdr))) {
    return false;
  }

  struct ether_hdr* eth = rte_pktmbuf_mtod(pkt, struct ether_hdr*);
  if (eth->ether_type != rte_cpu_to_be_16(NDN_ETHERTYPE)) {
    return false;
  }

  rte_pktmbuf_adj(pkt, sizeof(struct ether_hdr));

  return true;
}

static inline bool
RxFace_ProcessInterest(RxFace* face, struct rte_mbuf* pkt, TlvDecoder* d)
{
  Packet_SetNdnPktType(pkt, NdnPktType_Interest);
  InterestPkt* interest = Packet_GetInterestHdr(pkt);
  NdnError e = DecodeInterest(d, interest);

  bool ok = e == NdnError_OK;
  face->nInterestPkts += (int)ok;
  return ok;
}

static inline bool
RxFace_ProcessData(RxFace* face, struct rte_mbuf* pkt, TlvDecoder* d)
{
  Packet_SetNdnPktType(pkt, NdnPktType_Data);
  DataPkt* data = Packet_GetDataHdr(pkt);
  NdnError e = DecodeData(d, data);

  bool ok = e == NdnError_OK;
  face->nDataPkts += (int)ok;
  return ok;
}

static inline bool
RxFace_ProcessNetPkt(RxFace* face, struct rte_mbuf* pkt, TlvDecoder* d,
                     uint8_t firstOctet)
{
  if (firstOctet == TT_Interest) {
    return RxFace_ProcessInterest(face, pkt, d);
  }
  if (firstOctet == TT_Data) {
    return RxFace_ProcessData(face, pkt, d);
  }
  return false;
}

static inline struct rte_mbuf*
RxFace_ProcessLpPkt(RxFace* face, struct rte_mbuf* pkt, TlvDecoder* d)
{
  // To accommodate reassembly, this function may return a different (reassembled) rte_mbuf to
  // replace the input packet. The reassembler can also retain the LP fragment and return NULL.
  // This function internally frees invalid mbufs.

  Packet_SetL2PktType(pkt, L2PktType_NdnlpV2);
  LpPkt* lpp = Packet_GetLpHdr(pkt);
  NdnError e = DecodeLpPkt(d, lpp);
  if (unlikely(e != NdnError_OK)) {
    rte_pktmbuf_free(pkt);
    return NULL;
  }

  if (LpPkt_HasPayload(lpp)) {
    rte_pktmbuf_adj(pkt, lpp->payloadOff);

    if (LpPkt_IsFragmented(lpp)) {
      pkt = InOrderReassembler_Receive(&face->reassembler, pkt);
      if (pkt == NULL) {
        return NULL;
      }
      lpp = NULL; // received lpp does not apply to reassembled packet
    }

    TlvDecoder d1;
    MbufLoc_Init(&d1, pkt);
    bool res = RxFace_ProcessNetPkt(face, pkt, &d1, MbufLoc_PeekOctet(&d1));
    if (likely(res)) {
      return pkt;
    }
    rte_pktmbuf_free(pkt);
    return NULL;
  }

  rte_pktmbuf_free(pkt);
  return NULL;
}

uint16_t
RxFace_RxBurst(RxFace* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  uint16_t nReceived = rte_eth_rx_burst(face->port, face->queue, pkts, nPkts);

  for (uint16_t i = 0; i < nReceived; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    bool ok = RxFace_ProcessFrame(face, pkt);
    if (ok) {
      TlvDecoder d;
      MbufLoc_Init(&d, pkt);
      uint8_t firstOctet = MbufLoc_PeekOctet(&d);

      if (firstOctet == TT_LpPacket) {
        pkts[i] = RxFace_ProcessLpPkt(face, pkt, &d);
        ok = true;
      } else {
        ok = RxFace_ProcessNetPkt(face, pkt, &d, firstOctet);
      }
    }

    if (!ok) {
      rte_pktmbuf_free(pkt);
      pkts[i] = NULL;
    }
  }

  return nReceived;
}