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

void
MakeNack(Packet* npkt, NackReason reason)
{
  assert(Packet_GetL3PktType(npkt) == L3PktType_Interest);
  Packet_InitLpL3Hdr(npkt)->nackReason = reason;
  Packet_SetL3PktType(npkt, L3PktType_Nack);
}
