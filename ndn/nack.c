#include "nack.h"
#include "packet.h"

void
MakeNack(Packet* npkt, NackReason reason)
{
  assert(Packet_GetL3PktType(npkt) == L3PktType_Interest);
  Packet_InitLpL3Hdr(npkt)->nackReason = reason;
  Packet_SetL3PktType(npkt, L3PktType_Nack);
}
