#include "packet.h"
#include "tlv-decoder.h"

const char* PktType_Strings_[] = {
  [PktFragment] = "fragment", [PktInterest] = "interest", [PktSInterest] = "interest",
  [PktData] = "data",         [PktSData] = "data",        [PktNack] = "nack",
  [PktSNack] = "nack",
};

bool
Packet_Parse(Packet* npkt, ParseFor parseFor) {
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(pkt) && rte_mbuf_refcnt_read(pkt) == 1);
  pkt->packet_type = 0;
  PacketPriv* priv = Packet_GetPriv_(npkt);
  NDNDPDK_ASSERT(pkt->priv_size >= sizeof(*priv));
  POISON(priv);

  LpHeader* lph = &priv->lp;
  if (unlikely(!LpHeader_Parse(lph, pkt))) {
    return false;
  }

  if (lph->l2.fragCount > 1) {
    Packet_SetType(npkt, PktFragment);
    return true;
  }

  return Packet_ParseL3(npkt, parseFor);
}

bool
Packet_ParseL3(Packet* npkt, ParseFor parseFor) {
  PacketPriv* priv = Packet_GetPriv_(npkt);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(pkt));
  NDNDPDK_ASSERT(pkt->data_len >= 1);

  uint8_t type = rte_pktmbuf_mtod(pkt, const uint8_t*)[0];
  switch (type) {
    case TtInterest:
      Packet_SetType(npkt, priv->lpl3.nackReason == 0 ? PktInterest : PktNack);
      return PInterest_Parse(&priv->interest, pkt, parseFor);
    case TtData:
      Packet_SetType(npkt, PktData);
      return PData_Parse(&priv->data, pkt, parseFor);
  }
  return false;
}

__attribute__((nonnull, returns_nonnull)) static Packet*
Clone_Finish(const Packet* npkt, struct rte_mbuf* pkt) {
  Mbuf_SetTimestamp(pkt, Mbuf_GetTimestamp(Packet_ToMbuf(npkt)));
  Packet* output = Packet_FromMbuf(pkt);
  Packet_SetType(output, PktType_ToSlim(Packet_GetType(npkt)));
  *Packet_GetPriv_(output) = (const PacketPriv){0};
  return output;
}

__attribute__((nonnull)) static Packet*
Clone_Linear(Packet* npkt, PacketMempools* mp, PacketTxAlign align) {
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint32_t fragCount = SPDK_CEIL_DIV(pkt->pkt_len, align.fragmentPayloadSize);
  NDNDPDK_ASSERT(fragCount < LpMaxFragments);
  struct rte_mbuf* frames[LpMaxFragments];
  if (unlikely(rte_pktmbuf_alloc_bulk(mp->packet, frames, fragCount) != 0)) {
    return NULL;
  }

  TlvDecoder d = TlvDecoder_Init(pkt);
  uint32_t fragIndex = 0;
  frames[fragIndex]->data_off = RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom;
  TlvDecoder_Fragment(&d, d.length, frames, &fragIndex, fragCount, align.fragmentPayloadSize,
                      RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom);

  pkt = Mbuf_ChainVector(frames, fragCount);
  NDNDPDK_ASSERT(pkt != NULL);
  return Clone_Finish(npkt, pkt);
}

__attribute__((nonnull)) static Packet*
Clone_Chained(Packet* npkt, PacketMempools* mp) {
  struct rte_mbuf* header = rte_pktmbuf_alloc(mp->header);
  if (unlikely(header == NULL)) {
    return NULL;
  }
  header->data_off = RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom;

  struct rte_mbuf* payload = rte_pktmbuf_clone(Packet_ToMbuf(npkt), mp->indirect);
  if (unlikely(payload == NULL)) {
    rte_pktmbuf_free(header);
    return NULL;
  }
  if (unlikely(!Mbuf_Chain(header, header, payload))) {
    return NULL;
  }

  return Clone_Finish(npkt, header);
}

Packet*
Packet_Clone(Packet* npkt, PacketMempools* mp, PacketTxAlign align) {
  if (align.linearize) {
    return Clone_Linear(npkt, mp, align);
  }
  return Clone_Chained(npkt, mp);
}
