#include "packet.h"

const char*
PktType_ToString(PktType t)
{
  switch (t) {
    case PktFragment:
      return "fragment";
    case PktInterest:
    case PktSInterest:
      return "interest";
    case PktData:
    case PktSData:
      return "data";
    case PktNack:
    case PktSNack:
      return "nack";
    default:
      return "bad-PktType";
  }
}

bool
Packet_Parse(Packet* npkt)
{
  PacketPriv* priv = Packet_GetPriv_(npkt);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  pkt->packet_type = 0;

  LpHeader* lph = &priv->lp;
  if (unlikely(!LpHeader_Parse(lph, pkt))) {
    return false;
  }

  if (unlikely(pkt->pkt_len == 0)) {
    // there isn't any feature that depends on IDLE packets yet
    return false;
  }

  if (lph->l2.fragCount > 1) {
    // PktFragment is zero, no need to invoke setter
    NDNDPDK_ASSERT(Packet_GetType(npkt) == PktFragment);
    return true;
  }

  return Packet_ParseL3(npkt);
}

bool
Packet_ParseL3(Packet* npkt)
{
  PacketPriv* priv = Packet_GetPriv_(npkt);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  if (unlikely(pkt->data_len == 0)) {
    // TlvDecoder ensures there's no empty segment, so an empty first segment means an empty packet
    NDNDPDK_ASSERT(pkt->pkt_len == 0);
    return false;
  }

  uint8_t type = *rte_pktmbuf_mtod(pkt, const uint8_t*);
  switch (type) {
    case TtInterest:
      Packet_SetType(npkt, priv->lpl3.nackReason == 0 ? PktInterest : PktNack);
      return PInterest_Parse(&priv->interest, pkt);
    case TtData:
      Packet_SetType(npkt, PktData);
      return PData_Parse(&priv->data, pkt);
  }
  return false;
}

Packet*
Packet_Clone(Packet* npkt, struct rte_mempool* headerMp, struct rte_mempool* indirectMp)
{
  struct rte_mbuf* header = rte_pktmbuf_alloc(headerMp);
  if (unlikely(header == NULL)) {
    return NULL;
  }

  struct rte_mbuf* payload = rte_pktmbuf_clone(Packet_ToMbuf(npkt), indirectMp);
  if (unlikely(payload == NULL)) {
    rte_pktmbuf_free(header);
    return NULL;
  }
  if (unlikely(!Mbuf_Chain(header, header, payload))) {
    rte_pktmbuf_free(header);
    rte_pktmbuf_free(payload);
    return NULL;
  }

  Packet* output = Packet_FromMbuf(header);
  Packet_SetType(output, PktType_ToSlim(Packet_GetType(npkt)));
  *Packet_GetPriv_(output) = (const PacketPriv){ 0 };
  return output;
}
