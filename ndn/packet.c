#include "packet.h"

static const char* L3PktType_Strings[L3PktType_MAX] = {
  "none", "interest", "data", "nack",
};

const char*
L3PktType_ToString(L3PktType t)
{
  return L3PktType_Strings[t];
}

NdnError
Packet_ParseL2(Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  LpHeader* lph = __Packet_GetLpHdr(npkt);
  uint32_t payloadOff, tlvSize;
  NdnError e = LpHeader_FromPacket(lph, pkt, &payloadOff, &tlvSize);
  RETURN_IF_UNLIKELY_ERROR;
  Packet_SetL2PktType(npkt, L2PktType_NdnlpV2);

  if (unlikely(tlvSize < pkt->pkt_len)) { // strip Ethernet trailer
    assert(pkt->nb_segs == 1);
    pkt->pkt_len = tlvSize;
    pkt->data_len = (uint16_t)tlvSize;
  }
  Packet_Adj(pkt, payloadOff); // strip LpHeader
  return NdnError_OK;
}

NdnError
Packet_ParseL3(Packet* npkt, struct rte_mempool* nameMp)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  MbufLoc ml;
  MbufLoc_Init(&ml, pkt);
  switch (MbufLoc_PeekOctet(&ml)) {
    case TT_Interest: {
      NdnError e =
        PInterest_FromPacket(__Packet_GetInterestHdr(npkt), pkt, nameMp);
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
      NdnError e = PData_FromPacket(__Packet_GetDataHdr(npkt), pkt, nameMp);
      if (likely(e == NdnError_OK)) {
        Packet_SetL3PktType(npkt, L3PktType_Data);
      }
      return e;
    }
  }
  return NdnError_BadType;
}

Packet*
ClonePacket(Packet* npkt, struct rte_mempool* headerMp,
            struct rte_mempool* indirectMp)
{
  struct rte_mbuf* header = rte_pktmbuf_alloc(headerMp);
  if (unlikely(header == NULL)) {
    return NULL;
  }

  struct rte_mbuf* body = rte_pktmbuf_clone(Packet_ToMbuf(npkt), indirectMp);
  if (unlikely(body == NULL)) {
    rte_pktmbuf_free(header);
    return NULL;
  }
  rte_pktmbuf_chain(header, body);
  Packet* outNpkt = Packet_FromMbuf(header);

  // copy PacketPriv
  Packet_SetL2PktType(outNpkt, Packet_GetL2PktType(npkt));
  Packet_SetL3PktType(outNpkt, Packet_GetL3PktType(npkt));
  rte_memcpy(__Packet_GetPriv(outNpkt), __Packet_GetPriv(npkt),
             sizeof(PacketPriv));
  return outNpkt;
}
