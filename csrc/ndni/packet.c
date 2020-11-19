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

__attribute__((nonnull, returns_nonnull)) static Packet*
Packet_Clone_Finish(Packet* npkt, struct rte_mbuf* pkt)
{
  Packet* output = Packet_FromMbuf(pkt);
  Packet_SetType(output, PktType_ToSlim(Packet_GetType(npkt)));
  *Packet_GetPriv_(output) = (const PacketPriv){ 0 };
  return output;
}

__attribute__((nonnull)) static Packet*
Packet_Clone_Linear(Packet* npkt, PacketMempools* mp)
{
  struct rte_mbuf* m = rte_pktmbuf_alloc(mp->packet);
  if (unlikely(m == NULL)) {
    return NULL;
  }
  m->data_off = RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom;
  NDNDPDK_ASSERT(m->data_off <= m->buf_len);

  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint8_t* room = (uint8_t*)rte_pktmbuf_append(m, pkt->pkt_len);
  if (unlikely(room == NULL)) {
    rte_pktmbuf_free(m);
    return NULL;
  }

  Mbuf_CopyTo(pkt, room);
  return Packet_Clone_Finish(npkt, m);
}

__attribute__((nonnull)) static Packet*
Packet_Clone_Chained(Packet* npkt, PacketMempools* mp)
{
  struct rte_mbuf* header = rte_pktmbuf_alloc(mp->header);
  if (unlikely(header == NULL)) {
    return NULL;
  }

  struct rte_mbuf* payload = rte_pktmbuf_clone(Packet_ToMbuf(npkt), mp->indirect);
  if (unlikely(payload == NULL)) {
    rte_pktmbuf_free(header);
    return NULL;
  }
  if (unlikely(!Mbuf_Chain(header, header, payload))) {
    rte_pktmbuf_free(header);
    rte_pktmbuf_free(payload);
    return NULL;
  }

  return Packet_Clone_Finish(npkt, header);
}

Packet*
Packet_Clone(Packet* npkt, PacketMempools* mp, PacketTxAlign align)
{
  if (align.linearize) {
    return Packet_Clone_Linear(npkt, mp);
  }
  return Packet_Clone_Chained(npkt, mp);
}
