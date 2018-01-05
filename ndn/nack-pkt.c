#include "nack-pkt.h"
#include "packet.h"

void
MakeNack(struct rte_mbuf* pkt, NackReason reason)
{
  LpPkt* lpp;
  if (Packet_GetL2PktType(pkt) == L2PktType_NdnlpV2) {
    lpp = Packet_GetLpHdr(pkt);
  } else {
    Packet_SetL2PktType(pkt, L2PktType_NdnlpV2);
    lpp = Packet_GetLpHdr(pkt);
    memset(lpp, 0, sizeof(*lpp));
  }

  lpp->nackReason = reason;
  MbufLoc_Init(&lpp->payload, pkt);
}
