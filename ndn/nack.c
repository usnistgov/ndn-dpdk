#include "nack.h"
#include "packet.h"

void
MakeNack(struct rte_mbuf* pkt, NackReason reason)
{
  Packet* npkt = Packet_FromMbuf(pkt);
  LpL3* lpl3 = Packet_InitLpL3Hdr(npkt);
  lpl3->nackReason = reason;
  Packet_SetL3PktType(npkt, L3PktType_Nack);
}
