#include "packet.h"

NdnError
Packet_ParseL2(Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  LpHeader* lph = __Packet_GetLpHdr(npkt);
  uint32_t payloadOff;
  NdnError e = LpHeader_FromPacket(lph, pkt, &payloadOff);
  RETURN_IF_UNLIKELY_ERROR;
  Packet_SetL2PktType(npkt, L2PktType_NdnlpV2);

  Packet_Adj(pkt, payloadOff); // strip LpHeader
  return NdnError_OK;
}

NdnError
Packet_ParseL3(Packet* npkt, struct rte_mempool* mpName)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  MbufLoc ml;
  MbufLoc_Init(&ml, pkt);
  switch (MbufLoc_PeekOctet(&ml)) {
    case TT_Interest: {
      NdnError e =
        PInterest_FromPacket(__Packet_GetInterestHdr(npkt), pkt, mpName);
      if (likely(e == NdnError_OK)) {
        if (Packet_InitLpL3Hdr(npkt)->nackReason > 0) {
          Packet_SetL3PktType(npkt, L3PktType_Nack);
        } else {
          Packet_SetL3PktType(npkt, L3PktType_Interest);
        }
      }
      return e;
    }
    case TT_Data: {
      NdnError e = PData_FromPacket(__Packet_GetDataHdr(npkt), pkt, mpName);
      if (likely(e == NdnError_OK)) {
        Packet_SetL3PktType(npkt, L3PktType_Data);
      }
      return e;
    }
  }
  return NdnError_BadType;
}
