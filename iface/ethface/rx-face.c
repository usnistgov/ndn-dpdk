#include "rx-face.h"
#include "../../core/logger.h"

static inline bool
EthRxFace_ProcessFrame(EthRxFace* face, struct rte_mbuf* pkt)
{
  ++face->nFrames;

  if (unlikely(pkt->pkt_len < sizeof(struct ether_hdr))) {
    ZF_LOGD(_ETHFACE_LOG_PREFIX "len=%" PRIu32 " no-ether_hdr",
            _ETHFACE_LOG_PARAM, pkt->pkt_len);
    return false;
  }

  struct ether_hdr* eth = rte_pktmbuf_mtod(pkt, struct ether_hdr*);
  if (eth->ether_type != rte_cpu_to_be_16(NDN_ETHERTYPE)) {
    ZF_LOGD(_ETHFACE_LOG_PREFIX "ether_type=%" PRIX16 " not-NDN",
            _ETHFACE_LOG_PARAM, eth->ether_type);
    return false;
  }

  Packet_Adj(pkt, sizeof(struct ether_hdr));

  return true;
}

static inline bool
EthRxFace_ProcessInterest(EthRxFace* face, struct rte_mbuf* pkt, TlvDecoder* d)
{
  Packet_SetNdnPktType(pkt, NdnPktType_Interest);
  InterestPkt* interest = Packet_GetInterestHdr(pkt);
  NdnError e = DecodeInterest(d, interest);

  bool ok = e == NdnError_OK;
  if (unlikely(!ok)) {
    ZF_LOGD(_ETHFACE_LOG_PREFIX "interest-decode-error=%d", _ETHFACE_LOG_PARAM,
            e);
  }
  face->nInterestPkts += (int)ok;
  return ok;
}

static inline bool
EthRxFace_ProcessData(EthRxFace* face, struct rte_mbuf* pkt, TlvDecoder* d)
{
  Packet_SetNdnPktType(pkt, NdnPktType_Data);
  DataPkt* data = Packet_GetDataHdr(pkt);
  NdnError e = DecodeData(d, data);

  bool ok = e == NdnError_OK;
  if (unlikely(!ok)) {
    ZF_LOGD(_ETHFACE_LOG_PREFIX "data-decode-error=%d", _ETHFACE_LOG_PARAM, e);
  }
  face->nDataPkts += (int)ok;
  return ok;
}

static inline bool
EthRxFace_ProcessNetPkt(EthRxFace* face, struct rte_mbuf* pkt, TlvDecoder* d,
                        uint8_t firstOctet)
{
  if (firstOctet == TT_Interest) {
    return EthRxFace_ProcessInterest(face, pkt, d);
  }
  if (firstOctet == TT_Data) {
    return EthRxFace_ProcessData(face, pkt, d);
  }

  ZF_LOGD(_ETHFACE_LOG_PREFIX "unknown-net-type=%" PRIX8, _ETHFACE_LOG_PARAM,
          firstOctet);
  return false;
}

static inline struct rte_mbuf*
EthRxFace_ProcessLpPkt(EthRxFace* face, struct rte_mbuf* pkt, TlvDecoder* d)
{
  // To accommodate reassembly, this function may return a different (reassembled) rte_mbuf to
  // replace the input packet. The reassembler can also retain the LP fragment and return NULL.
  // This function internally frees invalid mbufs.

  Packet_SetL2PktType(pkt, L2PktType_NdnlpV2);
  LpPkt* lpp = Packet_GetLpHdr(pkt);
  NdnError e = DecodeLpPkt(d, lpp);
  if (unlikely(e != NdnError_OK)) {
    ZF_LOGD(_ETHFACE_LOG_PREFIX "lp-decode-error=%d", _ETHFACE_LOG_PARAM, e);
    rte_pktmbuf_free(pkt);
    return NULL;
  }

  if (LpPkt_HasPayload(lpp)) {
    Packet_Adj(pkt, lpp->payloadOff);

    if (LpPkt_IsFragmented(lpp)) {
      pkt = InOrderReassembler_Receive(&face->reassembler, pkt);
      if (pkt == NULL) {
        return NULL;
      }
      lpp = NULL; // received lpp does not apply to reassembled packet
    }

    TlvDecoder d1;
    MbufLoc_Init(&d1, pkt);
    bool res = EthRxFace_ProcessNetPkt(face, pkt, &d1, MbufLoc_PeekOctet(&d1));
    if (likely(res)) {
      return pkt;
    }
  } else {
    ZF_LOGD(_ETHFACE_LOG_PREFIX "no-payload", _ETHFACE_LOG_PARAM);
  }

  rte_pktmbuf_free(pkt);
  return NULL;
}

uint16_t
EthRxFace_RxBurst(EthRxFace* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  uint16_t nReceived = rte_eth_rx_burst(face->port, face->queue, pkts, nPkts);

  for (uint16_t i = 0; i < nReceived; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    bool ok = EthRxFace_ProcessFrame(face, pkt);
    if (ok) {
      TlvDecoder d;
      MbufLoc_Init(&d, pkt);
      uint8_t firstOctet = MbufLoc_PeekOctet(&d);

      if (firstOctet == TT_LpPacket) {
        pkts[i] = EthRxFace_ProcessLpPkt(face, pkt, &d);
        ok = true;
      } else {
        ok = EthRxFace_ProcessNetPkt(face, pkt, &d, firstOctet);
      }
    }

    if (!ok) {
      rte_pktmbuf_free(pkt);
      pkts[i] = NULL;
    }
  }

  return nReceived;
}