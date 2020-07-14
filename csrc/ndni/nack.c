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
Nack_FromInterest(Packet* npkt, NackReason reason)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(pkt) && rte_mbuf_refcnt_read(pkt) == 1);
  switch (Packet_GetType(npkt)) {
    case PktInterest:
      Packet_SetType(npkt, PktNack);
      break;
    case PktSInterest:
      Packet_SetType(npkt, PktSNack);
      break;
    default:
      NDNDPDK_ASSERT(false);
      break;
  }
  Packet_GetLpL3Hdr(npkt)->nackReason = reason;
  return npkt;
}
