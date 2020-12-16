#include "nack.h"
#include "packet.h"

const char*
NackReason_ToString(NackReason reason)
{
  switch (reason) {
    case NackCongestion:
      return "Congestion";
    case NackDuplicate:
      return "Duplicate";
    case NackNoRoute:
      return "NoRoute";
    case NackUnspecified:
      return "Unspecified";
    default:
      return "(other)";
  }
}

Packet*
Nack_FromInterest(Packet* npkt, NackReason reason, PacketMempools* mp, PacketTxAlign align)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(pkt));
  NDNDPDK_ASSERT(rte_mbuf_refcnt_read(pkt) == 1);
  NDNDPDK_ASSERT(PktType_ToSlim(Packet_GetType(npkt)) == PktSInterest);

  if (unlikely(rte_pktmbuf_headroom(pkt) < RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom ||
               (align.linearize &&
                (!rte_pktmbuf_is_contiguous(pkt) || pkt->pkt_len > align.fragmentPayloadSize)))) {
    LpL3 l3 = *Packet_GetLpL3Hdr(npkt);

    npkt = Packet_Clone(npkt, mp, align);
    rte_pktmbuf_free(pkt);
    NULLize(pkt);

    if (unlikely(npkt == NULL)) {
      return NULL;
    }
    *Packet_GetLpL3Hdr(npkt) = l3;
  }

  Packet_SetType(npkt, PktSNack);
  Packet_GetLpL3Hdr(npkt)->nackReason = reason;
  return npkt;
}
