#include "locator.h"

#define NDN_ETHERTYPE 0x8624
#define ETHER_VLAN_HLEN (RTE_ETHER_HDR_LEN + sizeof(struct rte_vlan_hdr))

static uint16_t
AppendEtherVlanHdr(uint8_t* buffer, const struct rte_ether_addr* src,
                   const struct rte_ether_addr* dst, uint16_t vid, uint16_t etherType)
{
  struct rte_ether_hdr* ether = (struct rte_ether_hdr*)buffer;
  rte_ether_addr_copy(dst, &ether->d_addr);
  rte_ether_addr_copy(src, &ether->s_addr);
  ether->ether_type = rte_cpu_to_be_16(etherType);
  if (vid == 0) {
    return RTE_ETHER_HDR_LEN;
  }

  ether->ether_type = rte_cpu_to_be_16(RTE_ETHER_TYPE_VLAN);
  struct rte_vlan_hdr* vlan = (struct rte_vlan_hdr*)RTE_PTR_ADD(buffer, RTE_ETHER_HDR_LEN);
  vlan->vlan_tci = rte_cpu_to_be_16(vid);
  vlan->eth_proto = rte_cpu_to_be_16(etherType);
  return ETHER_VLAN_HLEN;
}

static bool
EtherUnicast_Match(const uint8_t* buffer, const struct rte_mbuf* m)
{
  return m->data_len >= RTE_ETHER_HDR_LEN &&
         memcmp(rte_pktmbuf_mtod(m, const uint8_t*), buffer, RTE_ETHER_HDR_LEN) == 0;
}

static bool
VlanUnicast_Match(const uint8_t* buffer, const struct rte_mbuf* m)
{
  return m->data_len >= ETHER_VLAN_HLEN &&
         memcmp(rte_pktmbuf_mtod(m, const uint8_t*), buffer, ETHER_VLAN_HLEN) == 0;
}

static bool
EtherMulticast_Match(const uint8_t* buffer, const struct rte_mbuf* m)
{
  const struct rte_ether_hdr* ether = rte_pktmbuf_mtod(m, const struct rte_ether_hdr*);
  return m->data_len >= RTE_ETHER_HDR_LEN && ether->ether_type == rte_cpu_to_be_16(NDN_ETHERTYPE) &&
         rte_is_multicast_ether_addr(&ether->d_addr);
}

#define VLAN_MCAST_OFF (offsetof(struct rte_ether_hdr, ether_type))

static bool
VlanMulticast_Match(const uint8_t* buffer, const struct rte_mbuf* m)
{
  const struct rte_ether_hdr* ether = rte_pktmbuf_mtod(m, const struct rte_ether_hdr*);
  return m->data_len >= ETHER_VLAN_HLEN && rte_is_multicast_ether_addr(&ether->d_addr) &&
         memcmp(RTE_PTR_ADD(ether, VLAN_MCAST_OFF), RTE_PTR_ADD(buffer, VLAN_MCAST_OFF),
                ETHER_VLAN_HLEN - VLAN_MCAST_OFF) == 0;
}

EthRxMatch
EthLocator_MakeRxMatch(const EthLocator* loc, uint8_t* buffer)
{
  memset(buffer, 0, ETHHDR_BUFLEN);
  AppendEtherVlanHdr(buffer, &loc->remote, &loc->local, loc->vlan, NDN_ETHERTYPE);
  return rte_is_unicast_ether_addr(&loc->remote)
           ? (loc->vlan == 0 ? EtherUnicast_Match : VlanUnicast_Match)
           : (loc->vlan == 0 ? EtherMulticast_Match : VlanMulticast_Match);
}

void
EthLocator_MakeFlowPattern(const EthLocator* loc, EthFlowPattern* flow)
{
  memset(flow, 0, sizeof(*flow));
  size_t i = 0;

  memset(&flow->ethMask, 0xFF, sizeof(flow->ethMask));
  if (rte_is_unicast_ether_addr(&loc->remote)) {
    rte_ether_addr_copy(&loc->local, &flow->ethSpec.dst);
    rte_ether_addr_copy(&loc->remote, &flow->ethSpec.src);
  } else {
    memset(&flow->ethMask.src, 0x00, sizeof(flow->ethMask.src));
    rte_ether_addr_copy(&loc->remote, &flow->ethSpec.dst);
  }
  flow->ethSpec.type = rte_cpu_to_be_16(NDN_ETHERTYPE);

  flow->pattern[i].type = RTE_FLOW_ITEM_TYPE_ETH;
  flow->pattern[i].spec = &flow->ethSpec;
  flow->pattern[i].mask = &flow->ethMask;
  ++i;

  if (loc->vlan != 0) {
    flow->ethSpec.type = rte_cpu_to_be_16(RTE_ETHER_TYPE_VLAN);
    memset(&flow->vlanMask, 0xFF, sizeof(flow->vlanMask));
    flow->vlanSpec.tci = rte_cpu_to_be_16(loc->vlan);
    flow->vlanSpec.inner_type = rte_cpu_to_be_16(NDN_ETHERTYPE);

    flow->pattern[i].type = RTE_FLOW_ITEM_TYPE_VLAN;
    flow->pattern[i].spec = &flow->vlanSpec;
    flow->pattern[i].mask = &flow->vlanMask;
    ++i;
  }

  flow->pattern[i].type = RTE_FLOW_ITEM_TYPE_END;
  ++i;
  NDNDPDK_ASSERT(i <= RTE_DIM(flow->pattern));
}

uint16_t
EthLocator_MakeTxHdr(const EthLocator* loc, uint8_t* buffer)
{
  memset(buffer, 0, ETHHDR_BUFLEN);
  return AppendEtherVlanHdr(buffer, &loc->local, &loc->remote, loc->vlan, NDN_ETHERTYPE);
}
