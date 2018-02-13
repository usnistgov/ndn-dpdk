#include "packet.h"

NdnError
Packet_ParseL3(Packet* npkt, struct rte_mempool* mpName)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  NdnError e = NdnError_BadType;
  TlvDecodePos d0;
  MbufLoc_Init(&d0, pkt);
  switch (MbufLoc_PeekOctet(&d0)) {
    case TT_Interest:
      e = PInterest_FromPacket(__Packet_GetInterestHdr(npkt), pkt, mpName);
      if (likely(e == NdnError_OK)) {
        Packet_SetL3PktType(npkt, L3PktType_Interest);
      }
      break;
    case TT_Data:
      e = PData_FromPacket(__Packet_GetDataHdr(npkt), pkt, mpName);
      if (likely(e == NdnError_OK)) {
        Packet_SetL3PktType(npkt, L3PktType_Data);
      }
      break;
  }
  return e;
}
