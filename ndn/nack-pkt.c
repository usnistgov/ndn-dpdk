#include "nack-pkt.h"
#include "packet.h"

void
MakeNack(struct rte_mbuf* pkt, NackReason reason)
{
  Packet* npkt = Packet_FromMbuf(pkt);
  LpPkt* lpp;
  if (Packet_GetL2PktType(npkt) == L2PktType_NdnlpV2) {
    lpp = Packet_GetLpHdr(npkt);
  } else {
    Packet_SetL2PktType(npkt, L2PktType_NdnlpV2);
    lpp = Packet_GetLpHdr(npkt);
    memset(lpp, 0, sizeof(*lpp));
    MbufLoc_Init(&lpp->payload, pkt);
  }

  lpp->nackReason = reason;
  Packet_SetL3PktType(npkt, L3PktType_Nack);
}
