#include "gtpip.h"
#include "../core/logger.h"
#include "face.h"

N_LOG_INIT(EthGtpip);

typedef enum ExtractResult {
  ExtractResultNone,
  ExtractResultIPv4,
  ExtractResultFaceID,
} __rte_packed ExtractResult;

__attribute__((nonnull)) static __rte_always_inline uint64_t
ProcessBulk(EthGtpip* g, char dir, struct rte_mbuf* pkts[], uint32_t count,
            __attribute__((nonnull))
            ExtractResult extract(const struct rte_mbuf* pkt, uintptr_t* key),
            __attribute__((nonnull)) bool updatePkt(struct rte_mbuf* pkt, EthFacePriv* priv)) {
  NDNDPDK_ASSERT(count <= RTE_MIN_T(MaxBurstSize, RTE_HASH_LOOKUP_BULK_MAX, uint32_t));
  uintptr_t lookupKeys[RTE_HASH_LOOKUP_BULK_MAX];
  uint64_t lookupMask = 0;
  uint32_t nLookups = 0;
  FaceID faceIDs[RTE_HASH_LOOKUP_BULK_MAX];
  uint64_t faceMask = 0;
  uint32_t nFaces = 0;
  for (uint32_t i = 0; i < count; ++i) {
    uintptr_t key = 0;
    ExtractResult extracted = extract(pkts[i], &key);
    switch (extracted) {
      case ExtractResultNone:
        continue;
      case ExtractResultIPv4:
        rte_bit_set(&lookupMask, i);
        lookupKeys[nLookups++] = key;
        break;
      case ExtractResultFaceID:
        rte_bit_set(&faceMask, i);
        faceIDs[nFaces++] = key;
        break;
    }
  }
  if (nLookups + nFaces == 0) {
    N_LOGD("bulk-%c none-extracted count=%" PRIu32, dir, count);
    return 0;
  }

  uint64_t hitMask = 0;
  uintptr_t hitData[RTE_HASH_LOOKUP_BULK_MAX];
  if (nLookups > 0) {
    int nHits = rte_hash_lookup_bulk_data(g->ipv4, (const void**)lookupKeys, nLookups, &hitMask,
                                          (void**)hitData);
    if (unlikely(nHits < 0)) {
      N_LOGD("bulk-%c lookup-fail " N_LOG_ERROR_ERRNO, dir, nHits);
      return 0;
    }
  }

  uint64_t processedMask = 0;
  nLookups = 0;
  nFaces = 0;
  for (uint32_t i = 0; i < count; ++i) {
    FaceID id = 0;
    if (rte_bit_test(&lookupMask, i)) {
      uint32_t hitIndex = nLookups++;
      if (likely(rte_bit_test(&hitMask, hitIndex))) {
        id = (FaceID)hitData[hitIndex];
      } else {
        continue;
      }
    } else if (rte_bit_test(&faceMask, i)) {
      id = faceIDs[nFaces++];
    } else {
      continue;
    }

    EthFacePriv* priv = Face_GetPriv(Face_Get(id));
    if (likely(updatePkt(pkts[i], priv))) {
      rte_bit_set(&processedMask, i);
    }
  }

  N_LOGD("bulk-%c processed count=%" PRIu32 " lookups=%016" PRIx64 " faces=%016" PRIx64
         " processed=%016" PRIx64,
         dir, count, lookupMask, faceMask, processedMask);
  return processedMask;
}

__attribute__((nonnull)) static __rte_always_inline ExtractResult
DlExtractKey(const struct rte_mbuf* pkt, uintptr_t* key) {
  const struct rte_ether_hdr* eth = rte_pktmbuf_mtod(pkt, const struct rte_ether_hdr*);
  if (unlikely(pkt->data_len < RTE_ETHER_HDR_LEN + sizeof(struct rte_ipv4_hdr)) ||
      eth->ether_type != rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV4)) {
    return ExtractResultNone;
  }
  const struct rte_ipv4_hdr* ip = RTE_PTR_ADD(eth, RTE_ETHER_HDR_LEN);
  *key = (uintptr_t)&ip->dst_addr;
  return ExtractResultIPv4;
}

__attribute__((nonnull)) static __rte_always_inline bool
DlUpdatePkt(struct rte_mbuf* pkt, EthFacePriv* priv) {
  EthTxHdr_Prepend(&priv->txHdr, pkt, EthTxHdrFlagsGtpip);
  return true;
}

uint64_t
EthGtpip_ProcessDownlinkBulk(EthGtpip* g, struct rte_mbuf* pkts[], uint32_t count) {
  return ProcessBulk(g, 'D', pkts, count, DlExtractKey, DlUpdatePkt);
}

// Uplink header lengths, from outer Ethernet to inner IPv4.
enum {
  UlHdrLenBase = RTE_ETHER_HDR_LEN + sizeof(struct rte_udp_hdr) + sizeof(EthGtpHdr) +
                 sizeof(struct rte_ipv4_hdr),
  UlHdrLenIpv4 = UlHdrLenBase + sizeof(struct rte_ipv4_hdr),
  UlHdrLenVlanIpv4 = UlHdrLenIpv4 + sizeof(struct rte_vlan_hdr),
  UlHdrLenIpv6 = UlHdrLenBase + sizeof(struct rte_ipv6_hdr),
  UlHdrLenVlanIpv6 = UlHdrLenIpv6 + sizeof(struct rte_vlan_hdr),
};

__attribute__((nonnull)) static __rte_always_inline ExtractResult
UlExtractKey(const struct rte_mbuf* pkt, uintptr_t* key) {
  FaceID id = Mbuf_GetMark(pkt);
  if (id != 0) {
    *key = id;
    return ExtractResultFaceID;
  }

  const struct rte_ether_hdr* eth = rte_pktmbuf_mtod(pkt, const struct rte_ether_hdr*);
  const struct rte_vlan_hdr* vlan = RTE_PTR_ADD(eth, RTE_ETHER_HDR_LEN);
  uint16_t hdrLen = 0;
  if (likely(pkt->data_len >= UlHdrLenIpv4) &&
      eth->ether_type == rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV4)) {
    hdrLen = UlHdrLenIpv4;
  } else if (likely(pkt->data_len >= UlHdrLenVlanIpv4) &&
             eth->ether_type == rte_cpu_to_be_16(RTE_ETHER_TYPE_VLAN) &&
             vlan->eth_proto == rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV4)) {
    hdrLen = UlHdrLenVlanIpv4;
  } else if (likely(pkt->data_len >= UlHdrLenIpv6) &&
             eth->ether_type == rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV6)) {
    hdrLen = UlHdrLenIpv6;
  } else if (likely(pkt->data_len >= UlHdrLenVlanIpv6) &&
             eth->ether_type == rte_cpu_to_be_16(RTE_ETHER_TYPE_VLAN) &&
             vlan->eth_proto == rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV6)) {
    hdrLen = UlHdrLenVlanIpv6;
  } else {
    return ExtractResultNone;
  }

  const struct rte_ipv4_hdr* iip = RTE_PTR_ADD(eth, hdrLen - sizeof(*iip));
  const struct rte_udp_hdr* udp = RTE_PTR_SUB(iip, sizeof(*udp) + sizeof(EthGtpHdr));
  if (unlikely(udp->src_port != rte_cpu_to_be_16(RTE_GTPU_UDP_PORT)) ||
      unlikely(udp->dst_port != rte_cpu_to_be_16(RTE_GTPU_UDP_PORT))) {
    return ExtractResultNone;
  }

  *key = (uintptr_t)&iip->src_addr;
  return ExtractResultIPv4;
}

__attribute__((nonnull)) static __rte_always_inline bool
UlUpdatePkt(struct rte_mbuf* pkt, EthFacePriv* priv) {
  if (unlikely(!(EthRxMatch_Match(&priv->rxMatch, pkt) & EthRxMatchResultGtp))) {
    return false;
  }

  const struct rte_ether_hdr* eth = rte_pktmbuf_mtod(pkt, const struct rte_ether_hdr*);
  struct rte_ether_hdr* eth1 =
    (struct rte_ether_hdr*)rte_pktmbuf_adj(pkt, priv->rxMatch.len - sizeof(struct rte_udp_hdr) -
                                                  sizeof(struct rte_ipv4_hdr) - RTE_ETHER_HDR_LEN);
  eth1->dst_addr = eth->dst_addr; // TAP netif has same MAC address as physical EthDev
  eth1->src_addr = eth->src_addr;
  eth1->ether_type = rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV4);
  return true;
}

uint64_t
EthGtpip_ProcessUplinkBulk(EthGtpip* g, struct rte_mbuf* pkts[], uint32_t count) {
  return ProcessBulk(g, 'U', pkts, count, UlExtractKey, UlUpdatePkt);
}
