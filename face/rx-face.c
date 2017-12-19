#include "rx-face.h"
#include "packet.h"

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
  Packet_SetNdnNetType(pkt, NdnNetType_Interest);
  InterestPkt* interest = Packet_GetInterestHdr(pkt);
  NdnError e = DecodeInterest(d, interest);

  bool ok = e == NdnError_OK;
  face->nInterestPkts += (int)ok;
  return ok;
}

static inline bool
RxFace_ProcessData(RxFace* face, struct rte_mbuf* pkt, TlvDecoder* d)
{
  Packet_SetNdnNetType(pkt, NdnNetType_Data);
  DataPkt* data = Packet_GetDataHdr(pkt);
  NdnError e = DecodeData(d, data);

  bool ok = e == NdnError_OK;
  face->nDataPkts += (int)ok;
  return ok;
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

      switch (MbufLoc_PeekOctet(&d)) {
        case TT_Interest: {
          ok = RxFace_ProcessInterest(face, pkt, &d);
          break;
        }
        case TT_Data: {
          ok = RxFace_ProcessData(face, pkt, &d);
          break;
        }
      }
    }

    if (!ok) {
      rte_pktmbuf_free(pkt);
      pkts[i] = NULL;
    }
  }

  return nReceived;
}