#include "packet.h"

static const char* L3PktTypeStrings[L3PktTypeMAX] = {
  "none",
  "interest",
  "data",
  "nack",
};

const char*
L3PktTypeToString(L3PktType t)
{
  return L3PktTypeStrings[t];
}

NdnError
Packet_ParseL2(Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  LpHeader* lph = Packet_GetLpHdr_(npkt);
  uint32_t payloadOff, tlvSize;
  NdnError e = LpHeader_FromPacket(lph, pkt, &payloadOff, &tlvSize);
  RETURN_IF_ERROR;
  Packet_SetL2PktType(npkt, L2PktTypeNdnlpV2);

  if (unlikely(tlvSize < pkt->pkt_len)) { // strip Ethernet trailer
    assert(pkt->nb_segs == 1);
    pkt->pkt_len = tlvSize;
    pkt->data_len = (uint16_t)tlvSize;
  }
  Packet_Adj(pkt, payloadOff); // strip LpHeader
  return NdnErrOK;
}

NdnError
Packet_ParseL3(Packet* npkt, struct rte_mempool* nameMp)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  MbufLoc ml;
  MbufLoc_Init(&ml, pkt);
  switch (MbufLoc_PeekOctet(&ml)) {
    case TtInterest: {
      NdnError e =
        PInterest_FromPacket(Packet_GetInterestHdr_(npkt), pkt, nameMp);
      if (likely(e == NdnErrOK)) {
        if (Packet_InitLpL3Hdr(npkt)->nackReason > 0) {
          Packet_SetL3PktType(npkt, L3PktTypeNack);
        } else {
          Packet_SetL3PktType(npkt, L3PktTypeInterest);
        }
      }
      return e;
    }
    case TtData: {
      NdnError e = PData_FromPacket(Packet_GetDataHdr_(npkt), pkt, nameMp);
      if (likely(e == NdnErrOK)) {
        Packet_SetL3PktType(npkt, L3PktTypeData);
      }
      return e;
    }
  }
  return NdnErrBadType;
}

Packet*
ClonePacket(Packet* npkt,
            struct rte_mempool* headerMp,
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
  rte_memcpy(
    Packet_GetPriv_(outNpkt), Packet_GetPriv_(npkt), sizeof(PacketPriv));
  return outNpkt;
}
